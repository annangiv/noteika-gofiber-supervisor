package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"my-app/actor"
	"my-app/db"
	"my-app/supervisor"
	"my-app/web"
)

func main() {
	log.Println("🚀 Bootstrapping Keller (Go Fiber supervised node)...")

	// Resolve database path
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data"
	}

	// Initialize BadgerDB embedded storage
	repo, err := db.NewBadgerRepo(dbPath)
	if err != nil {
		log.Fatalf("[Main] FATAL: Failed to initialize BadgerDB: %v", err)
	}
	defer func() {
		log.Println("[Main] Closing BadgerDB storage...")
		if err := repo.Close(); err != nil {
			log.Printf("[Main] Error closing BadgerDB: %v", err)
		}
	}()

	// Initialize Actor Registry and Gateway
	registry := actor.NewActorRegistry()
	gateway := actor.NewActorGateway(registry)

	// Create Supervisor with the VaultActor factory
	sup := supervisor.NewSupervisor(registry, func() actor.Actor {
		return actor.NewVaultActor(repo)
	})

	// Configure supervisor to retry up to 5 times with exponential backoff
	sup.WithPolicy(supervisor.RestartMaxRetries, 5, 200*time.Millisecond, 5*time.Second)

	// Start supervisor monitoring in the background
	log.Println("[Main] Starting Supervisor...")
	sup.Start()
	defer sup.Stop()

	// Resolve server port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start Fiber web server
	srv := web.NewServer(gateway, sup, registry)
	
	go func() {
		if err := srv.Start(port); err != nil {
			log.Printf("[Main] Fiber web server stopped: %v", err)
		}
	}()

	// Handle OS interrupt signals for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[Main] Shutting down supervised node gracefully...")
}
