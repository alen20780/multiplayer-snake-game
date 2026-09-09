package main

import (
	"math/rand"
)

// Food represents an edible item placed in the game world
type Food struct {
	ID    int
	X     float64
	Y     float64
	Type  int // 0: regular, 1: super (from dead snake)
	Value int
}

// FoodManager handles food spawning and persistence
type FoodManager struct {
	foodItems map[int]*Food
	nextID    int
	config    Config
	rnd       *rand.Rand
}

// NewFoodManager initializes food items across the world
func NewFoodManager(cfg Config, rnd *rand.Rand) *FoodManager {
	fm := &FoodManager{
		foodItems: make(map[int]*Food),
		nextID:    1,
		config:    cfg,
		rnd:       rnd,
	}

	for i := 0; i < cfg.FoodCount; i++ {
		fm.SpawnRandomFood(0, cfg.FoodScoreBonus)
	}

	return fm
}

// SpawnRandomFood creates a food item at a random coordinate
func (fm *FoodManager) SpawnRandomFood(foodType, value int) *Food {
	padding := fm.config.FoodRadius * 4
	x := padding + fm.rnd.Float64()*(fm.config.WorldWidth-2*padding)
	y := padding + fm.rnd.Float64()*(fm.config.WorldHeight-2*padding)

	f := &Food{
		ID:    fm.nextID,
		X:     x,
		Y:     y,
		Type:  foodType,
		Value: value,
	}
	fm.foodItems[f.ID] = f
	fm.nextID++
	return f
}

// SpawnFoodAt creates a food item at a specific coordinate (e.g. from dead snakes)
func (fm *FoodManager) SpawnFoodAt(x, y float64, foodType, value int) *Food {
	// Clamp within boundaries
	padding := fm.config.FoodRadius * 2
	if x < padding {
		x = padding
	} else if x > fm.config.WorldWidth-padding {
		x = fm.config.WorldWidth - padding
	}
	if y < padding {
		y = padding
	} else if y > fm.config.WorldHeight-padding {
		y = fm.config.WorldHeight - padding
	}

	f := &Food{
		ID:    fm.nextID,
		X:     x,
		Y:     y,
		Type:  foodType,
		Value: value,
	}
	fm.foodItems[f.ID] = f
	fm.nextID++
	return f
}

// RemoveFood removes a consumed food item and optionally replenishes
func (fm *FoodManager) RemoveFood(id int) {
	delete(fm.foodItems, id)
}

// Replenish ensures the world maintains at least cfg.FoodCount normal foods
func (fm *FoodManager) Replenish() {
	currentCount := len(fm.foodItems)
	deficit := fm.config.FoodCount - currentCount
	for i := 0; i < deficit; i++ {
		fm.SpawnRandomFood(0, fm.config.FoodScoreBonus)
	}
}

// ToDTO returns network-ready representations of food
func (fm *FoodManager) ToDTO() []FoodDTO {
	list := make([]FoodDTO, 0, len(fm.foodItems))
	for _, f := range fm.foodItems {
		list = append(list, FoodDTO{
			ID:    f.ID,
			X:     f.X,
			Y:     f.Y,
			Type:  f.Type,
			Value: f.Value,
		})
	}
	return list
}
