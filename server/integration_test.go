package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestLiveMultiplayerWebSocket(t *testing.T) {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}

	// Connect Client 1
	c1, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Client 1 failed to connect: %v", err)
	}
	defer c1.Close()

	// Connect Client 2
	c2, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Client 2 failed to connect: %v", err)
	}
	defer c2.Close()

	// Read Client 1 welcome message
	var welcome WelcomeMessage
	_, msgBytes, err := c1.ReadMessage()
	if err != nil {
		t.Fatalf("Client 1 read error: %v", err)
	}
	if err := json.Unmarshal(msgBytes, &welcome); err != nil {
		t.Fatalf("Unmarshal welcome error: %v", err)
	}
	if welcome.Type != MsgTypeWelcome || welcome.PlayerID == "" {
		t.Fatalf("Invalid welcome message: %+v", welcome)
	}

	// Send Join for both players
	c1.WriteJSON(ClientMessage{Type: MsgTypeJoin, Name: "PlayerAlpha"})
	c2.WriteJSON(ClientMessage{Type: MsgTypeJoin, Name: "PlayerBeta"})

	// Verify that state messages arrive and contain 2 players
	receivedTwoPlayers := false
	timeout := time.After(3 * time.Second)

	for !receivedTwoPlayers {
		select {
		case <-timeout:
			t.Fatalf("Timed out waiting for state with 2 players")
		default:
			c1.SetReadDeadline(time.Now().Add(1 * time.Second))
			_, data, err := c1.ReadMessage()
			if err != nil {
				continue
			}
			var state StateMessage
			if err := json.Unmarshal(data, &state); err == nil && state.Type == MsgTypeState {
				if len(state.Snakes) == 2 {
					receivedTwoPlayers = true
					fmt.Printf("[Test] Verified 2 players in authoritative state snapshot: %s and %s\n",
						state.Snakes[0].Name, state.Snakes[1].Name)
				}
			}
		}
	}

	// Send steer input from Client 1
	c1.WriteJSON(ClientMessage{Type: MsgTypeInput, Direction: DirUp})
	log.Println("[Test] Live WebSocket verification complete.")
}
