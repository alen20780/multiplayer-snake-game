package main

import "time"

// Config holds all configurable game mechanics and server constants
type Config struct {
	WorldWidth         float64
	WorldHeight        float64
	TickRate           int           // Ticks per second (e.g. 60)
	TickInterval       time.Duration // Duration per tick
	SnakeSpeed         float64       // Distance units moved per second
	SegmentSpacing     float64       // Distance between snake body segments
	InitialLength      int           // Initial number of segments
	SnakeRadius        float64       // Collision radius of head/segments
	FoodCount          int           // Number of persistent food items
	FoodRadius         float64       // Collision radius of food
	FoodScoreBonus     int           // Score gained per normal food
	MaxPlayers         int           // Maximum simultaneous players
	DeadDropPercentage float64       // Ratio of dead snake segments converted to food
}

// DefaultConfig provides recommended game settings
var DefaultConfig = Config{
	WorldWidth:         5000.0,
	WorldHeight:        5000.0,
	TickRate:           60,
	TickInterval:       time.Second / 60,
	SnakeSpeed:         300.0, // 300 units per sec = 5 units per tick at 60 TPS
	SegmentSpacing:     16.0,  // Stored historical trail spacing
	InitialLength:      12,    // Starts with 12 segments
	SnakeRadius:        14.0,  // Radius for collision detection
	FoodCount:          600,   // Plentiful food over 5000x5000 world
	FoodRadius:         8.0,   // Radius of food item
	FoodScoreBonus:     10,    // Points per regular food
	MaxPlayers:         200,   // Cap simultaneous players
	DeadDropPercentage: 0.6,   // Drop 60% of body as food upon death
}
