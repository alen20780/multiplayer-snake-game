package main

import (
	"math"
)

// CollisionResult holds what collision occurred for a snake
type CollisionResult struct {
	HitWall     bool
	HitSelf     bool
	HitSnakeID  string
	HeadToHead  bool
	EatenFoodID int
}

// CheckWallCollision checks if snake head is outside the world bounds
func CheckWallCollision(head Point, cfg Config) bool {
	radius := cfg.SnakeRadius
	if head.X-radius < 0 || head.X+radius > cfg.WorldWidth {
		return true
	}
	if head.Y-radius < 0 || head.Y+radius > cfg.WorldHeight {
		return true
	}
	return false
}

// CheckSelfCollision checks if snake head collided with its own body segments
func CheckSelfCollision(segments []Point, cfg Config) bool {
	if len(segments) <= 4 {
		return false
	}
	head := segments[0]
	// Skip neck segments (1, 2, 3) to allow smooth turning without false self-collision
	for i := 4; i < len(segments); i++ {
		seg := segments[i]
		dist := math.Hypot(head.X-seg.X, head.Y-seg.Y)
		if dist < cfg.SnakeRadius*1.5 {
			return true
		}
	}
	return false
}

// CheckSnakeBodyCollision checks if snake head collided with another snake's body
func CheckSnakeBodyCollision(head Point, otherSegments []Point, cfg Config) bool {
	// Skip head of other snake (which is checked separately for head-to-head)
	for i := 1; i < len(otherSegments); i++ {
		seg := otherSegments[i]
		dist := math.Hypot(head.X-seg.X, head.Y-seg.Y)
		if dist < cfg.SnakeRadius*1.75 {
			return true
		}
	}
	return false
}

// CheckHeadToHeadCollision checks if two snake heads collided
func CheckHeadToHeadCollision(head1, head2 Point, cfg Config) bool {
	dist := math.Hypot(head1.X-head2.X, head1.Y-head2.Y)
	return dist < cfg.SnakeRadius*2.0
}

// CheckFoodCollision returns the ID of any food item touched by the head
func CheckFoodCollision(head Point, foodItems map[int]*Food, cfg Config) *Food {
	touchRadius := cfg.SnakeRadius + cfg.FoodRadius
	touchRadiusSq := touchRadius * touchRadius

	for _, f := range foodItems {
		dx := head.X - f.X
		dy := head.Y - f.Y
		if (dx*dx + dy*dy) <= touchRadiusSq {
			return f
		}
	}
	return nil
}
