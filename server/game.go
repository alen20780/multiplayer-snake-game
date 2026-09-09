package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// PlayerInput holds queued direction requests from clients
type PlayerInput struct {
	PlayerID  string
	Direction string
}

// JoinRequest is sent when a player wants to join/respawn
type JoinRequest struct {
	Client   *Client
	Name     string
	RespChan chan *Player
}

// LeaveRequest is sent when a client disconnects
type LeaveRequest struct {
	PlayerID string
}

// Game owns the authoritative game state and runs the 60 TPS simulation loop
type Game struct {
	config      Config
	players     map[string]*Player
	foodManager *FoodManager
	tick        uint64
	rnd         *rand.Rand
	colorIdx    int

	// Thread-safe channels for game loop communication
	joinChan  chan JoinRequest
	leaveChan chan LeaveRequest
	inputChan chan PlayerInput

	// Hub for broadcasting state
	hub *Hub

	// State snapshot cache
	stateJSON atomic.Value

	// Mutex only used for shutdown/status queries, game state is mutated solely in Tick()
	mu     sync.RWMutex
	stopCh chan struct{}
}

// NewGame initializes a new authoritative game server
func NewGame(cfg Config, hub *Hub) *Game {
	source := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(source)

	g := &Game{
		config:      cfg,
		players:     make(map[string]*Player),
		foodManager: NewFoodManager(cfg, rnd),
		tick:        0,
		rnd:         rnd,
		joinChan:    make(chan JoinRequest, 64),
		leaveChan:   make(chan LeaveRequest, 64),
		inputChan:   make(chan PlayerInput, 512),
		hub:         hub,
		stopCh:      make(chan struct{}),
	}

	return g
}

// Start launches the authoritative 60 TPS game loop in a dedicated goroutine
func (g *Game) Start() {
	ticker := time.NewTicker(g.config.TickInterval)
	dt := g.config.TickInterval.Seconds()

	log.Printf("[GameServer] Game simulation started at %d TPS (World: %.0fx%.0f, Food: %d)",
		g.config.TickRate, g.config.WorldWidth, g.config.WorldHeight, g.config.FoodCount)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-g.stopCh:
				log.Println("[GameServer] Game simulation stopped")
				return
			case <-ticker.C:
				g.tickSimulation(dt)
			}
		}
	}()
}

// Stop signals the game loop to terminate
func (g *Game) Stop() {
	close(g.stopCh)
}

// RequestJoin requests to spawn or respawn a player
func (g *Game) RequestJoin(c *Client, name string) *Player {
	respChan := make(chan *Player, 1)
	g.joinChan <- JoinRequest{
		Client:   c,
		Name:     name,
		RespChan: respChan,
	}
	return <-respChan
}

// RequestLeave notifies the game loop that a player has disconnected
func (g *Game) RequestLeave(playerID string) {
	g.leaveChan <- LeaveRequest{PlayerID: playerID}
}

// QueueInput sends a player's direction change into the game loop
func (g *Game) QueueInput(playerID string, direction string) {
	select {
	case g.inputChan <- PlayerInput{PlayerID: playerID, Direction: direction}:
	default:
		// Queue full, drop input to prevent backpressure
	}
}

// tickSimulation executes a single frame of the authoritative simulation
func (g *Game) tickSimulation(dt float64) {
	g.tick++

	// 1. Process Joins
	g.processJoins()

	// 2. Process Leaves
	g.processLeaves()

	// 3. Process Queued Inputs
	g.processInputs()

	// 4. Update Snake Positions
	for _, p := range g.players {
		if p.Snake != nil && p.Snake.Alive {
			p.Snake.Move(dt, g.config)
		}
	}

	// 5. Detect and Resolve Collisions & Check Win/Fill Canvas Conditions
	g.resolveCollisions()

	// 6. Replenish Food
	g.foodManager.Replenish()

	// 7. Build and Broadcast Snapshot (only if clients are listening)
	if g.hub.ClientCount() > 0 {
		g.broadcastState()
	}
}

func (g *Game) processJoins() {
	for {
		select {
		case req := <-g.joinChan:
			playerID := req.Client.id
			name := req.Name
			if name == "" {
				name = fmt.Sprintf("Snake #%s", playerID[:4])
			}

			// Clean up previous snake if respawning
			if existing, ok := g.players[playerID]; ok && existing.Snake != nil && existing.Snake.Alive {
				g.killSnake(existing.Snake, "respawn")
			}

			color := PickSnakeColor(g.colorIdx)
			g.colorIdx++

			// Find valid spawn point away from world edges
			padding := 400.0
			spawnX := padding + g.rnd.Float64()*(g.config.WorldWidth-2*padding)
			spawnY := padding + g.rnd.Float64()*(g.config.WorldHeight-2*padding)

			dirs := []string{DirUp, DirDown, DirLeft, DirRight}
			initialDir := dirs[g.rnd.Intn(len(dirs))]

			snake := NewSnake(playerID, name, color, Point{X: spawnX, Y: spawnY}, initialDir, g.config)
			player := &Player{
				ID:     playerID,
				Name:   name,
				Color:  color,
				Snake:  snake,
				Client: req.Client,
			}
			g.players[playerID] = player

			log.Printf("[GameServer] Player '%s' (ID: %s) joined at (%.0f, %.0f)", name, playerID, spawnX, spawnY)
			req.RespChan <- player
		default:
			return
		}
	}
}

func (g *Game) processLeaves() {
	for {
		select {
		case req := <-g.leaveChan:
			if p, ok := g.players[req.PlayerID]; ok {
				if p.Snake != nil && p.Snake.Alive {
					g.killSnake(p.Snake, "disconnected")
				}
				delete(g.players, req.PlayerID)
				log.Printf("[GameServer] Player '%s' (ID: %s) left the game", p.Name, req.PlayerID)
				g.checkLastPlayerWin("Only remaining player standing - Victory!")
			}
		default:
			return
		}
	}
}

func (g *Game) processInputs() {
	for {
		select {
		case input := <-g.inputChan:
			if p, ok := g.players[input.PlayerID]; ok && p.Snake != nil && p.Snake.Alive {
				p.Snake.SetDirection(input.Direction)
			}
		default:
			return
		}
	}
}

func (g *Game) checkLastPlayerWin(reason string) {
	// Count how many players started/joined vs how many are currently alive
	alivePlayers := make([]*Player, 0)
	for _, p := range g.players {
		if p.Snake != nil && p.Snake.Alive {
			alivePlayers = append(alivePlayers, p)
		}
	}

	// If there were multiple participants and exactly 1 alive snake remains
	if len(alivePlayers) == 1 && len(g.players) > 1 {
		winner := alivePlayers[0]
		log.Printf("[GameServer] Player '%s' WON as the last player standing!", winner.Name)
		winner.Snake.Alive = false
		if winner.Client != nil {
			winner.Client.SendGameOver(winner.Snake.Score, reason, true)
		}
	}
}

func (g *Game) resolveCollisions() {
	deadSnakes := make(map[string]string) // snakeID -> reason

	// Pre-calculate segments for all alive snakes
	segmentsMap := make(map[string][]Point)
	for id, p := range g.players {
		if p.Snake != nil && p.Snake.Alive {
			segmentsMap[id] = p.Snake.GetSegments(g.config)
		}
	}

	// 1. Wall Collisions
	for id, p := range g.players {
		if p.Snake == nil || !p.Snake.Alive {
			continue
		}
		if CheckWallCollision(p.Snake.Head, g.config) {
			deadSnakes[id] = "Hit the electric boundary"
		}
	}

	// 2. Self Collisions
	for id, p := range g.players {
		if p.Snake == nil || !p.Snake.Alive {
			continue
		}
		if _, alreadyDead := deadSnakes[id]; alreadyDead {
			continue
		}
		if CheckSelfCollision(segmentsMap[id], g.config) {
			deadSnakes[id] = "Collided with own body"
		}
	}

	// 3. Head-to-Head & Snake-to-Snake collisions
	aliveIDs := make([]string, 0, len(segmentsMap))
	for id := range segmentsMap {
		if _, dead := deadSnakes[id]; !dead {
			aliveIDs = append(aliveIDs, id)
		}
	}

	for i := 0; i < len(aliveIDs); i++ {
		idA := aliveIDs[i]
		snakeA := g.players[idA].Snake

		for j := i + 1; j < len(aliveIDs); j++ {
			idB := aliveIDs[j]
			snakeB := g.players[idB].Snake

			// Check head-to-head
			if CheckHeadToHeadCollision(snakeA.Head, snakeB.Head, g.config) {
				// Both die in head-to-head impact
				deadSnakes[idA] = fmt.Sprintf("Head-on collision with %s", snakeB.Name)
				deadSnakes[idB] = fmt.Sprintf("Head-on collision with %s", snakeA.Name)
			}
		}
	}

	// Check Snake A head hitting Snake B body
	for _, idA := range aliveIDs {
		if _, dead := deadSnakes[idA]; dead {
			continue
		}
		snakeA := g.players[idA].Snake

		for _, idB := range aliveIDs {
			if idA == idB {
				continue
			}
			segsB := segmentsMap[idB]
			if CheckSnakeBodyCollision(snakeA.Head, segsB, g.config) {
				deadSnakes[idA] = fmt.Sprintf("Crashed into %s", g.players[idB].Name)
				// Reward killer with score bonus
				if g.players[idB].Snake != nil && g.players[idB].Snake.Alive {
					g.players[idB].Snake.Score += 50
				}
				break
			}
		}
	}

	// Apply deaths and drop food
	hadDeaths := len(deadSnakes) > 0
	for id, reason := range deadSnakes {
		if p, ok := g.players[id]; ok && p.Snake != nil && p.Snake.Alive {
			g.killSnake(p.Snake, reason)
			if p.Client != nil {
				p.Client.SendGameOver(p.Snake.Score, reason, false)
			}
		}
	}

	// If deaths occurred, check if only one survivor remains (Last Snake Standing Win)
	if hadDeaths {
		g.checkLastPlayerWin("All opponents eliminated - Victory!")
	}

	// 4. Food Collisions & Max Canvas Fill Condition
	for _, p := range g.players {
		if p.Snake == nil || !p.Snake.Alive {
			continue
		}
		food := CheckFoodCollision(p.Snake.Head, g.foodManager.foodItems, g.config)
		if food != nil {
			p.Snake.Grow(food.Value)
			g.foodManager.RemoveFood(food.ID)

			// Check if snake length has filled the canvas / reached victory limit
			if g.config.MaxCanvasLength > 0 && p.Snake.Length >= g.config.MaxCanvasLength {
				log.Printf("[GameServer] Snake '%s' filled the canvas (Length: %d)! VICTORY!", p.Name, p.Snake.Length)
				p.Snake.Alive = false
				if p.Client != nil {
					p.Client.SendGameOver(p.Snake.Score, fmt.Sprintf("Snake filled the entire canvas! (Max length %d achieved)", g.config.MaxCanvasLength), true)
				}
			}
		}
	}
}

func (g *Game) killSnake(s *Snake, reason string) {
	s.Alive = false
	log.Printf("[GameServer] Snake '%s' (Score: %d) died: %s", s.Name, s.Score, reason)

	// Drop dead snake segments as bonus food (super food type 1, 20 score)
	segs := s.GetSegments(g.config)
	step := 2
	for i := 0; i < len(segs); i += step {
		// Drop a fraction based on config
		if g.rnd.Float64() < g.config.DeadDropPercentage {
			g.foodManager.SpawnFoodAt(segs[i].X, segs[i].Y, 1, 20)
		}
	}
}

func (g *Game) broadcastState() {
	snakes := make([]SnakeDTO, 0, len(g.players))
	for _, p := range g.players {
		if p.Snake != nil {
			snakes = append(snakes, p.Snake.ToDTO(g.config))
		}
	}

	msg := StateMessage{
		Type:        MsgTypeState,
		Tick:        g.tick,
		PlayerCount: len(g.players),
		Snakes:      snakes,
		Food:        g.foodManager.ToDTO(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[GameServer] JSON marshal error: %v", err)
		return
	}

	g.hub.Broadcast(data)
}
