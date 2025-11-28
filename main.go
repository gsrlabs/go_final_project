// main.go
package main

import (

	"log"
	"net/http"
	"os"
	"todo/pkg/api"
	"todo/pkg/db"

	"github.com/joho/godotenv"
)

// main is the entry point of the Todo Scheduler application
// It loads configuration, initializes database and starts HTTP server
func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get server port from environment or use default
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	// Get database file path from environment or use default
	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		dbFile = "internal/db/scheduler.db"
	}

	if err := db.Init(dbFile); err != nil {
		log.Fatal("Database initialization error:", err)
	}

	// Configure logger to show timestamp and file location
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("INFO: Server starting on port %s", port)
	log.Printf("INFO: Open http://localhost:%s in your browser", port)

	err := http.ListenAndServe(":"+port, api.Router())
	if err != nil {
		log.Fatal("Server startup error:", err)
	}
}
