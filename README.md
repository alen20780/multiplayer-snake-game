# CYBERSNAKE - Real-Time Authoritative Multiplayer Snake Game

A high-performance full-stack real-time multiplayer Snake game built with **Go** and **HTML5 Canvas**. The Go server runs an authoritative game simulation at a fixed **60 ticks per second (TPS)** over WebSockets.

---

## Architecture Overview

```text
                 ┌─────────────────────────────────────────┐
                 │           Go Game Server                │
                 │                                         │
Players ────────►│   Gorilla WebSocket Hub (ReadPump)      │
                 │                    │                    │
                 │                    ▼                    │
                 │          Channel Event Queue            │
                 │      (joinChan, leaveChan, inputChan)   │
                 │                    │                    │
                 │                    ▼                    │
                 │     Authoritative Simulation Loop       │
                 │              (60 TPS)                   │
                 │     - Process user steering inputs      │
                 │     - Integrate continuous velocities   │
                 │     - Multi-tier collision engine       │
                 │     - Food consumption & replenishing   │
                 │                    │                    │
                 │                    ▼                    │
                 │       Broadcast Channel (non-blocking)  │
                 │                    │                    │
                 │                    ▼                    │
                 │   Gorilla WebSocket Hub (WritePump)     │
                 └────────────────────┬────────────────────┘
                                      │
                             Binary/JSON WebSocket
                                      │
                                      ▼
                 ┌─────────────────────────────────────────┐
                 │          HTML5 Canvas Client            │
                 │                                         │
                 │ - Smooth camera tracking player snake   │
                 │ - requestAnimationFrame render loop     │
                 │ - Independent refresh rate decoupling   │
                 │ - Minimap radar & live leaderboard      │
                 │ - Neon arcade HUD                       │
                 └─────────────────────────────────────────┘
```

---

## Technologies Used

- **Backend**:
  - **Go** (Standard Library `net/http`, `sync`, `time`)
  - **`github.com/gorilla/websocket`** for resilient bidirectional communication
  - Goroutines + Channels for lock-minimized concurrency
- **Frontend**:
  - **HTML5 Canvas API**
  - **Vanilla JavaScript (ES6+)**
  - **CSS3** Glassmorphism & Cyber/Neon styling
  - Native **WebSocket API**

---

## Folder Structure

```text
snake-game/
├── server/
│   ├── config.go        # World size, speed, tick rate, radii, food limits
│   ├── protocol.go      # JSON protocol structs (join, input, state, game_over)
│   ├── food.go          # Food manager, random spawning & drops
│   ├── snake.go         # Snake entity, continuous trail sampling & direction locks
│   ├── collision.go     # Wall, self, body, and head-on collision checks
│   ├── player.go        # Player session & color palette manager
│   ├── game.go          # Authoritative 60 TPS simulation loop & state broadcast
│   ├── websocket.go     # Hub, Client ReadPump & WritePump with drop protection
│   ├── main.go          # Server entrypoint & static file server
│   └── game_test.go     # Comprehensive unit tests
├── client/
│   ├── index.html       # Single-page UI with HUD, overlays & radar
│   ├── style.css        # Cyberpunk / arcade aesthetics
│   └── game.js          # Canvas rendering, camera follow & input dispatch
├── go.mod
├── go.sum
└── README.md
```

---

## WebSocket Networking Protocol

### Client → Server

1. **Join Arena**:
```json
{
  "type": "join",
  "name": "PilotX"
}
```

2. **Input Steering**:
```json
{
  "type": "input",
  "direction": "up" // "up" | "down" | "left" | "right"
}
```

3. **Ping (Heartbeat)**:
```json
{
  "type": "ping"
}
```

### Server → Client

1. **Welcome (On connection)**:
```json
{
  "type": "welcome",
  "playerId": "1725867291029",
  "worldWidth": 5000,
  "worldHeight": 5000,
  "tickRate": 60
}
```

2. **State Snapshot (60 times/sec)**:
```json
{
  "type": "state",
  "tick": 3482,
  "playerCount": 2,
  "snakes": [
    {
      "id": "1725867291029",
      "name": "PilotX",
      "color": "#00ff88",
      "x": 2520,
      "y": 1400,
      "direction": "right",
      "body": [{"x": 2520, "y": 1400}, {"x": 2504, "y": 1400}],
      "score": 80,
      "alive": true
    }
  ],
  "food": [
    {"id": 1, "x": 1200, "y": 800, "type": 0, "value": 10}
  ]
}
```

3. **Game Over (Upon collision)**:
```json
{
  "type": "game_over",
  "finalScore": 120,
  "reason": "Crashed into Viper"
}
```

---

## How the Authoritative Game Loop Works

1. A dedicated goroutine runs `time.NewTicker(time.Second / 60)`.
2. Every tick:
   - Drains queued **join requests** and initializes snakes at safe coordinates.
   - Drains queued **leave/disconnect requests** and reaps dead snakes.
   - Drains user **steering commands**, validating against 180° reverse turns.
   - Integrates snake head movement (`Speed * dt`) and records trail history.
   - Samples body segments uniformly along the historical trail.
   - Executes collision checks:
     - **Wall collision**: Out of `[0, WorldWidth] x [0, WorldHeight]` => Snake dies.
     - **Self collision**: Head within body segments (ignoring neck) => Snake dies.
     - **Body collision**: Head intersects another snake body segment => Snake dies, killer gains +50 score bonus.
     - **Head-to-head collision**: Both snakes impact heads directly => Both die.
     - **Food collision**: Head touches food => Snake length & score increase, food re-spawns elsewhere.
   - Upon death, the snake drops glowing super-food pellets along its corpse.
   - Serializes world state into JSON and broadcasts to all connected client write buffers.

---

## Network Performance & Concurrency Features

- **Decoupled Read & Write Pumps**: Each WebSocket connection maintains independent read and write goroutines with write timeouts (`5s`) and heartbeat pong timeouts (`60s`).
- **Slow Client Protection**: If a client's outgoing buffer fills up due to high network lag, the server drops the slow client instead of stalling the game loop.
- **Micro-Batching**: Multiple queued state frames are consolidated into single network writes with newline delimiters, minimizing system call overhead.
- **Untrusted Client Inputs**: Clients cannot transmit coordinates, speeds, lengths, or scores. The server is 100% authoritative.

---

## How to Run the Server

### Prerequisites
- Go 1.20 or newer

### 1. Install Dependencies & Build
```bash
cd /path/to/snake-game
go mod tidy
```

### 2. Run the Server
```bash
go run ./server
```
The server will start at:
```text
http://localhost:8080
```

### 3. Run Unit Tests
```bash
go test -v ./server
```

---

## Connecting Multiple Players Locally

1. Open your browser and navigate to:
   ```text
   http://localhost:8080
   ```
2. Enter your pilot codename and click **ENTER ARENA**.
3. Open a second browser window (or tab, or incognito mode) at `http://localhost:8080`.
4. Enter another name (e.g. "Snake #2") and click **ENTER ARENA**.
5. Both snakes will now appear in the shared 5000x5000 arena with real-time updates!

---

## Configuration Settings (`server/config.go`)

| Parameter | Default | Description |
|-----------|---------|-------------|
| `WorldWidth` | 5000 | Arena width in units |
| `WorldHeight` | 5000 | Arena height in units |
| `TickRate` | 60 | Simulation updates per second |
| `SnakeSpeed` | 300 | Units moved per second |
| `SegmentSpacing` | 16 | Pixel gap between body segments |
| `InitialLength` | 12 | Segments spawned with |
| `SnakeRadius` | 14 | Collision hit circle |
| `FoodCount` | 600 | Active food items distributed on map |
| `MaxPlayers` | 200 | Recommended connection ceiling |
