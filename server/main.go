package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	defaultPort := os.Getenv("PORT")
	if defaultPort == "" {
		defaultPort = "8080"
	}

	port := flag.String("port", defaultPort, "HTTP and WebSocket server port")
	flag.Parse()

	cfg := DefaultConfig
	hub := NewHub()
	game := NewGame(cfg, hub)

	go hub.Run()
	game.Start()

	// WebSocket handler
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, game, w, r)
	})

	// Static file handler for client files
	fileServer := http.FileServer(http.Dir("./client"))
	http.Handle("/", fileServer)

	serverAddr := fmt.Sprintf(":%s", *port)
	srv := &http.Server{
		Addr: serverAddr,
	}

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("=========================================================")
		log.Printf(" Snake Server running on http://localhost:%s", *port)
		log.Printf(" World: %.0f x %.0f | Ticks: %d TPS | Food: %d items",
			cfg.WorldWidth, cfg.WorldHeight, cfg.TickRate, cfg.FoodCount)
		log.Printf("=========================================================")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-stop
	log.Println("\nShutting down server gracefully...")
	game.Stop()
	srv.Close()
	log.Println("Server stopped")
}
