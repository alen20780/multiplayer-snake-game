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
                 │     - Last Player Remaining Victory     │
                 │     - Canvas-Fill Max Length Victory    │
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
                 │ - Neon arcade HUD (Length/Capacity)     │
                 │ - Victory & Game Over dynamic overlays  │
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
  - **CSS3** Glassmorphism & Cyber/Neon styling with Golden Victory Theme
  - Native **WebSocket API**

---

## Victory & End-Game Conditions

1. **Last Player Remaining Victory**:
   - In multiplayer matches (2+ players), when opponents disconnect or crash, the sole remaining snake is declared the **Winner** with a victory broadcast (`type: "victory"`).
2. **Canvas-Fill Max Length Victory**:
   - When a player consumes enough food to reach the canvas fill capacity limit (`MaxCanvasLength`, default 200 segments), the game concludes with a **Canvas-Fill Victory**, celebrating the pilot's triumph!

---

## Folder Structure

```text
snake-game/
├── server/
│   ├── config.go          # World size, speed, tick rate, radii, food limits, max canvas length
│   ├── protocol.go        # JSON protocol structs (join, input, state, game_over, victory)
│   ├── food.go            # Food manager, random spawning & drops
│   ├── snake.go           # Snake entity, continuous trail sampling & direction locks
│   ├── collision.go       # Wall, self, body, and head-on collision checks
│   ├── player.go          # Player session & color palette manager
│   ├── game.go            # Authoritative 60 TPS simulation loop, win conditions & state broadcast
│   ├── websocket.go       # Hub, Client ReadPump & WritePump with drop protection & victory messaging
│   ├── main.go            # Server entrypoint & static file server
│   ├── game_test.go       # Comprehensive unit tests (movement, collisions, win conditions)
│   └── integration_test.go# Real-time WebSocket multi-client integration test
├── client/
│   ├── index.html         # Single-page UI with HUD, victory/death modal & radar
│   ├── style.css          # Cyberpunk / arcade aesthetics & golden victory card styling
│   └── game.js            # Canvas rendering, camera follow, win modal handler & radar
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
  "tickRate": 60,
  "maxCanvasLength": 200
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
  "reason": "Crashed into Viper",
  "won": false
}
```

4. **Victory (Last standing or canvas filled)**:
```json
{
  "type": "victory",
  "finalScore": 340,
  "reason": "All opponents eliminated - Victory!",
  "won": true
}
```

---

## How to Run the Server

### 1. Install Dependencies & Build
```bash
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

### 3. Run All Tests
```bash
go test -v ./server
```

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
| `MaxCanvasLength` | 200 | Snake length required to fill canvas and trigger victory |
