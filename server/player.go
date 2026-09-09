package main

// Player wraps player identity, connection handler, and snake entity
type Player struct {
	ID     string
	Name   string
	Color  string
	Snake  *Snake
	Client *Client
}

// Pre-curated palette of distinct vibrant neon/arcade colors for snakes
var SnakeColors = []string{
	"#00ff88", // Neon Mint
	"#ff0055", // Cyber Pink
	"#00d4ff", // Electric Cyan
	"#ffe600", // Bright Yellow
	"#b026ff", // Vivid Purple
	"#ff7700", // Neon Orange
	"#39ff14", // Acid Green
	"#ff3399", // Hot Magenta
	"#00e5ff", // Aqua Blue
	"#e0e0e0", // Chrome Silver
}

// PickSnakeColor deterministically picks a color based on player count or index
func PickSnakeColor(index int) string {
	return SnakeColors[index%len(SnakeColors)]
}
