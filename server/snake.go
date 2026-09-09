package main

import (
	"math"
)

// Allowed directions
const (
	DirUp    = "up"
	DirDown  = "down"
	DirLeft  = "left"
	DirRight = "right"
)

// Snake represents an active snake entity controlled by a player
type Snake struct {
	ID        string
	Name      string
	Color     string
	Head      Point
	Direction string
	NextDir   string // Queued direction for next tick
	Trail     []Point // Historical points for segment trailing
	Length    int     // Number of segments
	Score     int
	Alive     bool
	Speed     float64
}

// NewSnake creates a newly spawned snake
func NewSnake(id, name, color string, spawn Point, dir string, cfg Config) *Snake {
	if dir == "" {
		dir = DirRight
	}

	// Compute initial trail going backwards opposite of direction
	trail := make([]Point, 0, cfg.InitialLength*4)
	dx, dy := getDirectionVector(dir)

	// Fill trail history so body segments can be placed immediately behind head
	trailCount := cfg.InitialLength * int(cfg.SegmentSpacing)
	for i := 0; i <= trailCount; i++ {
		trail = append(trail, Point{
			X: spawn.X - float64(i)*dx,
			Y: spawn.Y - float64(i)*dy,
		})
	}

	return &Snake{
		ID:        id,
		Name:      name,
		Color:     color,
		Head:      spawn,
		Direction: dir,
		NextDir:   dir,
		Trail:     trail,
		Length:    cfg.InitialLength,
		Score:     0,
		Alive:     true,
		Speed:     cfg.SnakeSpeed,
	}
}

// SetDirection queues a direction change, disallowing direct 180-degree reverses
func (s *Snake) SetDirection(dir string) bool {
	if !s.Alive {
		return false
	}
	switch dir {
	case DirUp:
		if s.Direction == DirDown {
			return false
		}
	case DirDown:
		if s.Direction == DirUp {
			return false
		}
	case DirLeft:
		if s.Direction == DirRight {
			return false
		}
	case DirRight:
		if s.Direction == DirLeft {
			return false
		}
	default:
		return false
	}
	s.NextDir = dir
	return true
}

// Move advances the snake's head forward according to its velocity and tick delta
func (s *Snake) Move(dt float64, cfg Config) {
	if !s.Alive {
		return
	}

	s.Direction = s.NextDir
	dx, dy := getDirectionVector(s.Direction)
	distance := s.Speed * dt

	s.Head.X += dx * distance
	s.Head.Y += dy * distance

	// Prepend new head position to trail history
	s.Trail = append([]Point{s.Head}, s.Trail...)

	// Trim trail length to maximum needed for body segment reconstruction
	maxTrailNeeded := int(float64(s.Length+1)*cfg.SegmentSpacing) + 50
	if len(s.Trail) > maxTrailNeeded {
		s.Trail = s.Trail[:maxTrailNeeded]
	}
}

// GetSegments calculates the positions of all body segments sampled from the trail
func (s *Snake) GetSegments(cfg Config) []Point {
	segments := make([]Point, 0, s.Length)
	if len(s.Trail) == 0 {
		return segments
	}

	// First segment is the head
	segments = append(segments, s.Head)

	accumDist := 0.0
	targetSpacing := cfg.SegmentSpacing
	currentIdx := 0

	for i := 1; i < s.Length; i++ {
		targetDist := float64(i) * targetSpacing
		// Advance through trail until we reach target distance
		for currentIdx+1 < len(s.Trail) && accumDist < targetDist {
			p1 := s.Trail[currentIdx]
			p2 := s.Trail[currentIdx+1]
			segDist := math.Hypot(p2.X-p1.X, p2.Y-p1.Y)
			if accumDist+segDist >= targetDist {
				// Interpolate between p1 and p2
				ratio := (targetDist - accumDist) / segDist
				interpPoint := Point{
					X: p1.X + (p2.X-p1.X)*ratio,
					Y: p1.Y + (p2.Y-p1.Y)*ratio,
				}
				segments = append(segments, interpPoint)
				break
			}
			accumDist += segDist
			currentIdx++
		}
		// If trail ends early (rare), repeat last position
		if len(segments) <= i && len(s.Trail) > 0 {
			segments = append(segments, s.Trail[len(s.Trail)-1])
		}
	}

	return segments
}

// Grow increases length and score
func (s *Snake) Grow(value int) {
	s.Score += value
	// Grow 1 segment for each 10 score points
	s.Length += 1
}

// ToDTO converts internal snake state to network DTO
func (s *Snake) ToDTO(cfg Config) SnakeDTO {
	return SnakeDTO{
		ID:        s.ID,
		Name:      s.Name,
		Color:     s.Color,
		X:         s.Head.X,
		Y:         s.Head.Y,
		Direction: s.Direction,
		Body:      s.GetSegments(cfg),
		Score:     s.Score,
		Alive:     s.Alive,
	}
}

func getDirectionVector(dir string) (float64, float64) {
	switch dir {
	case DirUp:
		return 0, -1
	case DirDown:
		return 0, 1
	case DirLeft:
		return -1, 0
	case DirRight:
		return 1, 0
	default:
		return 1, 0
	}
}
