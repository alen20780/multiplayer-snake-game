package main

import (
	"math/rand"
	"testing"
	"time"
)

func TestSnakeMovement(t *testing.T) {
	cfg := DefaultConfig
	spawn := Point{X: 1000, Y: 1000}
	snake := NewSnake("p1", "TestSnake", "#ffffff", spawn, DirRight, cfg)

	dt := 1.0 / 60.0
	expectedMove := cfg.SnakeSpeed * dt

	snake.Move(dt, cfg)

	if snake.Head.X != spawn.X+expectedMove {
		t.Errorf("Expected head X to be %f, got %f", spawn.X+expectedMove, snake.Head.X)
	}
	if snake.Head.Y != spawn.Y {
		t.Errorf("Expected head Y to be %f, got %f", spawn.Y, snake.Head.Y)
	}
}

func TestDirectionReversalPrevention(t *testing.T) {
	cfg := DefaultConfig
	snake := NewSnake("p1", "TestSnake", "#ffffff", Point{X: 500, Y: 500}, DirRight, cfg)

	// 180 degree reverse should fail
	if snake.SetDirection(DirLeft) {
		t.Errorf("Expected 180-degree turn from Right to Left to be prevented")
	}

	// 90 degree turn should succeed
	if !snake.SetDirection(DirUp) {
		t.Errorf("Expected 90-degree turn from Right to Up to succeed")
	}
	if snake.NextDir != DirUp {
		t.Errorf("Expected NextDir to be Up, got %s", snake.NextDir)
	}

	// Now move to apply Up direction
	snake.Move(1.0/60.0, cfg)
	if snake.Direction != DirUp {
		t.Errorf("Expected Direction to be Up after Move, got %s", snake.Direction)
	}

	// Down should now be prevented
	if snake.SetDirection(DirDown) {
		t.Errorf("Expected 180-degree turn from Up to Down to be prevented")
	}
}

func TestWallCollision(t *testing.T) {
	cfg := DefaultConfig

	// Inside bounds
	inside := Point{X: 2500, Y: 2500}
	if CheckWallCollision(inside, cfg) {
		t.Errorf("Point inside world incorrectly reported as wall collision")
	}

	// Out of bounds - Left
	leftOut := Point{X: cfg.SnakeRadius - 1, Y: 1000}
	if !CheckWallCollision(leftOut, cfg) {
		t.Errorf("Expected wall collision for X < radius")
	}

	// Out of bounds - Right
	rightOut := Point{X: cfg.WorldWidth - cfg.SnakeRadius + 1, Y: 1000}
	if !CheckWallCollision(rightOut, cfg) {
		t.Errorf("Expected wall collision for X > WorldWidth - radius")
	}

	// Out of bounds - Top
	topOut := Point{X: 1000, Y: cfg.SnakeRadius - 1}
	if !CheckWallCollision(topOut, cfg) {
		t.Errorf("Expected wall collision for Y < radius")
	}

	// Out of bounds - Bottom
	bottomOut := Point{X: 1000, Y: cfg.WorldHeight - cfg.SnakeRadius + 1}
	if !CheckWallCollision(bottomOut, cfg) {
		t.Errorf("Expected wall collision for Y > WorldHeight - radius")
	}
}

func TestSelfCollision(t *testing.T) {
	cfg := DefaultConfig
	segments := []Point{
		{X: 100, Y: 100}, // Head
		{X: 100, Y: 116}, // Seg 1
		{X: 100, Y: 132}, // Seg 2
		{X: 100, Y: 148}, // Seg 3
		{X: 116, Y: 148}, // Seg 4
		{X: 116, Y: 100}, // Seg 5 (near head!)
	}

	// seg 5 is within radius*1.5 of head (dist is 16 < 14*1.5=21)
	if !CheckSelfCollision(segments, cfg) {
		t.Errorf("Expected self collision when body loops near head")
	}

	// Straight line should have no self collision
	straightSegments := []Point{
		{X: 100, Y: 100},
		{X: 100, Y: 116},
		{X: 100, Y: 132},
		{X: 100, Y: 148},
		{X: 100, Y: 164},
		{X: 100, Y: 180},
	}
	if CheckSelfCollision(straightSegments, cfg) {
		t.Errorf("Straight snake should not trigger self collision")
	}
}

func TestSnakeBodyCollision(t *testing.T) {
	cfg := DefaultConfig
	headA := Point{X: 500, Y: 500}

	otherSnakeSegments := []Point{
		{X: 600, Y: 500}, // Other head (ignored by body collision check)
		{X: 505, Y: 500}, // Other body segment 1 (dist=5 < 14*1.75=24.5)
		{X: 450, Y: 500},
	}

	if !CheckSnakeBodyCollision(headA, otherSnakeSegments, cfg) {
		t.Errorf("Expected snake head to collide with other snake body")
	}

	distantHead := Point{X: 200, Y: 200}
	if CheckSnakeBodyCollision(distantHead, otherSnakeSegments, cfg) {
		t.Errorf("Distant head should not collide with other snake body")
	}
}

func TestHeadToHeadCollision(t *testing.T) {
	cfg := DefaultConfig
	head1 := Point{X: 1000, Y: 1000}
	head2 := Point{X: 1010, Y: 1000} // dist = 10 < 28

	if !CheckHeadToHeadCollision(head1, head2, cfg) {
		t.Errorf("Expected head-to-head collision within 2*radius")
	}

	farHead := Point{X: 1050, Y: 1000} // dist = 50 > 28
	if CheckHeadToHeadCollision(head1, farHead, cfg) {
		t.Errorf("Far heads should not collide")
	}
}

func TestFoodConsumptionAndGrowth(t *testing.T) {
	cfg := DefaultConfig
	rnd := rand.New(rand.NewSource(42))
	fm := NewFoodManager(cfg, rnd)

	// Clear food and spawn one at fixed coordinate
	fm.foodItems = make(map[int]*Food)
	food := fm.SpawnFoodAt(1000, 1000, 0, 15)

	snake := NewSnake("p1", "Hero", "#00ff88", Point{X: 1002, Y: 1000}, DirRight, cfg)
	initialLen := snake.Length

	foundFood := CheckFoodCollision(snake.Head, fm.foodItems, cfg)
	if foundFood == nil {
		t.Fatalf("Expected snake to collide with food at (1000, 1000)")
	}
	if foundFood.ID != food.ID {
		t.Errorf("Expected food ID %d, got %d", food.ID, foundFood.ID)
	}

	snake.Grow(foundFood.Value)
	fm.RemoveFood(foundFood.ID)

	if snake.Score != 15 {
		t.Errorf("Expected score 15, got %d", snake.Score)
	}
	if snake.Length != initialLen+1 {
		t.Errorf("Expected length %d, got %d", initialLen+1, snake.Length)
	}
	if len(fm.foodItems) != 0 {
		t.Errorf("Expected food to be removed from manager")
	}
}

func TestFoodReplenish(t *testing.T) {
	cfg := DefaultConfig
	cfg.FoodCount = 50
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	fm := NewFoodManager(cfg, rnd)

	if len(fm.foodItems) != 50 {
		t.Errorf("Expected 50 food items initially, got %d", len(fm.foodItems))
	}

	// Remove 10 items
	count := 0
	for id := range fm.foodItems {
		fm.RemoveFood(id)
		count++
		if count >= 10 {
			break
		}
	}

	if len(fm.foodItems) != 40 {
		t.Errorf("Expected 40 food items after deletion, got %d", len(fm.foodItems))
	}

	fm.Replenish()
	if len(fm.foodItems) != 50 {
		t.Errorf("Expected replenish to bring count back to 50, got %d", len(fm.foodItems))
	}
}
