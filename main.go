// main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"todo/pkg/api"
	"todo/pkg/db"

	"github.com/joho/godotenv"
)


func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	dbFile := os.Getenv("TODO_DBFILE")
	if dbFile == "" {
		dbFile = "internal/db/scheduler.db"
	}

	if err := db.Init(dbFile); err != nil {
		log.Fatal("Ошибка инициализации базы данных:", err)
	}

	fmt.Printf("Сервер запущен на порту: %s\n", port)
	fmt.Println("Открыть: http://localhost:" + port)

	err := http.ListenAndServe(":"+port, api.Router())
	if err != nil {
		log.Fatal(err)
	}
}