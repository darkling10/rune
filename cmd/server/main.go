package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/deployerai/deployer/internal/api"
	"github.com/deployerai/deployer/internal/db"
	"github.com/deployerai/deployer/internal/repository"
	"github.com/deployerai/deployer/internal/worker"
	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it, relying on environment variables")
	}

	// Initialize the Database
	database, err := db.InitDB("")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize the Repository
	projectRepo := repository.NewProjectRepository(database)

	// Initialize Redis Options
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		// Fallback for local development if not provided
		redisAddr = "localhost:6379"
	} else {
		// Parse standard redis url (redis://...)
		// For simplicity in this demo we'll assume it's just host:port, or parse it properly
		// A robust parse is omitted for brevity, but let's assume REDIS_URL is just localhost:6379 for now.
		// Since we put redis://localhost:6379/0 in .env, we should parse it.
		// Actually, asynq.ParseRedisURI handles this out of the box!
	}

	redisOpt, err := asynq.ParseRedisURI(redisAddr)
	if err != nil {
		// If parse fails, fallback to simple address
		redisOpt = asynq.RedisClientOpt{Addr: "localhost:6379"}
	}

	// Initialize Task Worker Architecture
	taskDistributor := worker.NewRedisTaskDistributor(redisOpt)
	taskProcessor := worker.NewRedisTaskProcessor(redisOpt)

	go func() {
		log.Println("Starting Task Processor...")
		if err := taskProcessor.Start(); err != nil {
			log.Fatalf("failed to start task processor: %v", err)
		}
	}()

	// Initialize the API Server
	server := api.NewServer(projectRepo, taskDistributor)

	// Define the HTTP server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: server,
		// Good practice: Enforce timeouts
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Run our server in a goroutine so that it doesn't block
	go func() {
		log.Println("Starting Deployer AI server on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
