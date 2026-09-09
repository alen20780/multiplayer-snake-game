package main

// Client-to-Server message types
const (
	MsgTypeJoin  = "join"
	MsgTypeInput = "input"
	MsgTypePing  = "ping"
)

// Server-to-Client message types
const (
	MsgTypeWelcome  = "welcome"
	MsgTypeState    = "state"
	MsgTypeGameOver = "game_over"
	MsgTypePong     = "pong"
)

// ClientMessage represents an incoming payload from client
type ClientMessage struct {
	Type      string `json:"type"`
	Name      string `json:"name,omitempty"`
	Direction string `json:"direction,omitempty"`
}

// Point represents a 2D coordinate in the world
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// SnakeDTO is the network representation of a player's snake
type SnakeDTO struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Color     string  `json:"color"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Direction string  `json:"direction"`
	Body      []Point `json:"body"`
	Score     int     `json:"score"`
	Alive     bool    `json:"alive"`
}

// FoodDTO is the network representation of an edible item
type FoodDTO struct {
	ID    int     `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Type  int     `json:"type"`
	Value int     `json:"value"`
}

// WelcomeMessage sent to client immediately on connection or join
type WelcomeMessage struct {
	Type        string  `json:"type"`
	PlayerID    string  `json:"playerId"`
	WorldWidth  float64 `json:"worldWidth"`
	WorldHeight float64 `json:"worldHeight"`
	TickRate    int     `json:"tickRate"`
}

// StateMessage broadcasts world snapshot at 60 TPS
type StateMessage struct {
	Type        string     `json:"type"`
	Tick        uint64     `json:"tick"`
	PlayerCount int        `json:"playerCount"`
	Snakes      []SnakeDTO `json:"snakes"`
	Food        []FoodDTO  `json:"food"`
}

// GameOverMessage informs player of their death and final stats
type GameOverMessage struct {
	Type       string `json:"type"`
	FinalScore int    `json:"finalScore"`
	Reason     string `json:"reason"`
}
