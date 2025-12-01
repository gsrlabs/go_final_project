package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"todo/pkg/api"
	"todo/pkg/db"

	"github.com/joho/godotenv"
)

// main is the entry point of the Todo Scheduler application
// It loads configuration, initializes database and starts HTTP server
func main() {

	// For local development - will silently fail in Docker if .env doesn't exist
	if err := godotenv.Load(); err != nil {
		log.Printf("INFO: .env file not found, using environment variables")
	}

	// Get server port from environment or use default
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	// Get database file path from environment or use default
	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		dbFile = "data/scheduler.db"
	}

	// Debug: print all environment variables
	log.Printf("DEBUG: TODO_PORT=%s", os.Getenv("TODO_PORT"))
	log.Printf("DEBUG: TODO_DBFILE=%s", os.Getenv("TODO_DBFILE"))
	log.Printf("DEBUG: TODO_PASSWORD set=%t", os.Getenv("TODO_PASSWORD") != "")

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dbFile), 0755); err != nil {
		log.Printf("WARN: Failed to create data directory: %v", err)
	}

	// Create storage
	storage, err := db.NewStorage(dbFile)
	if err != nil {
		log.Fatal("Database initialization error:", err)
	}
	defer storage.Close()

	// Create API
	app := api.NewAPI(storage)

	// Configure logger to show timestamp and file location
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("INFO: Server starting on port %s", port)
	log.Printf("INFO: Database file: %s", dbFile)
	log.Printf("INFO: Open http://localhost:%s in your browser", port)

	err = http.ListenAndServe(":"+port, app.Router())
	if err != nil {
		log.Fatal("Server startup error:", err)
	}
}
