package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"time"
	"todo/pkg/db"
	"todo/pkg/model"

	"github.com/go-chi/chi/v5"
)

const dateLayout = "20060102"
const limit int = 50

// Router
func Router() http.Handler {
	r := chi.NewRouter()

	r.Handle("/*", http.FileServer(http.Dir("./web")))

	 // Публичные маршруты (не требуют аутентификации)
	r.Group(func(r chi.Router) {
        r.Get("/api/nextdate", nextDayHandler)
        r.Post("/api/signin", SignInHandler)
    })

	// Защищенные маршруты (требуют аутентификации)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/api/task", addTaskHandler)
		r.Get("/api/task", getTaskHandler)
		r.Put("/api/task", updateTaskHandler)
		r.Get("/api/tasks", tasksHandler)
		r.Post("/api/task/done", doneTaskHandler)
		r.Delete("/api/task", deleteTaskHandler)
	})

	return r
}

// authMiddleware - адаптер для chi
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        AuthMiddleware(next.ServeHTTP)(w, r)
    })
}

// nextDayHandler
func nextDayHandler(w http.ResponseWriter, r *http.Request) {
	nowStart := r.URL.Query().Get("now")
	dstart := r.URL.Query().Get("date")
	repeat := r.URL.Query().Get("repeat")

	if dstart == "" || repeat == "" {
		http.Error(w, `{"error":"date and repeat are required"}`, http.StatusBadRequest)
		return
	}

	var now time.Time
	if nowStart == "" {
		now = time.Now()
	} else {
		parsed, err := time.Parse(dateLayout, nowStart)
		if err != nil {
			http.Error(w, `{"error":"invalid now format"}`, http.StatusBadRequest)
			return
		}
		now = parsed
	}

	next, err := NextDate(now, dstart, repeat)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(next))
}

// addTaskHandler — POST /api/task
func addTaskHandler(w http.ResponseWriter, r *http.Request) {
	var input model.CreateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if input.Title == "" {
		sendError(w, "the title is empty", http.StatusBadRequest)
		return
	}

	date, err := normalizeDate(input.Date, input.Repeat)
	if err != nil {
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	task := model.Task{
		Date:    date,
		Title:   input.Title,
		Comment: input.Comment,
		Repeat:  input.Repeat,
	}

	id, err := db.Store.AddTask(&task)
	if err != nil {
		sendError(w, "saving error", http.StatusInternalServerError)
		return
	}

	sendJSON(w, map[string]any{"id": fmt.Sprintf("%d", id)})
}

// normalizeDate
func normalizeDate(dateStart, repeat string) (string, error) {
	now := time.Now()
	today := now.Format(dateLayout)

	if dateStart == "" || dateStart == "today" {
		dateStart = today
	}

	parsed, err := time.Parse(dateLayout, dateStart)
	if err != nil {
		return "", fmt.Errorf("invalid date format")
	}

	if parsed.Format(dateLayout) >= today {
		return dateStart, nil
	}

	if repeat != "" {
		next, err := NextDate(now, dateStart, repeat)
		if err != nil {
			return "", err
		}
		return next, nil
	}

	if parsed.Format(dateLayout) < today {
		return today, nil
	}

	return dateStart, nil
}

// getTasksHandler
func tasksHandler(w http.ResponseWriter, r *http.Request) {

	search := r.URL.Query().Get("search")
	var tasks db.TasksResp
	var err error

	if search == "" {
		tasks, err = db.Store.GetTasks(limit)
	} else {
		if date, err2 := time.Parse("02.01.2006", search); err2 == nil {
			y, m, d := date.Date()
			date := fmt.Sprintf("%d%02d%02d", y, m, d)
			tasks, err = db.Store.GetTasksByDate(limit, date)
		} else {
			tasks, err = db.Store.GetTasksByTitle(limit, search)
		}
	}

	if err != nil {
		sendError(w, "failed to get tasks", http.StatusInternalServerError)
		return
	}

	sendJSON(w, tasks)
}

// getTaskHandler
func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	if id == "" {
		http.Error(w, `{"error": "id not specified"}`, http.StatusBadRequest)
		return
	}

	task, err := db.Store.GetTask(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			sendError(w, "task not found", http.StatusNotFound)
		} else {
			sendError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	sendJSON(w, task)
}

// updateTaskHandler
func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var input model.TaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		sendError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if input.ID == "" {
		sendError(w, "id is required", http.StatusBadRequest)
		return
	}

	if input.Title == "" {
		sendError(w, "the title is empty", http.StatusBadRequest)
		return
	}

	date, err := normalizeDate(input.Date, input.Repeat)
	if err != nil {
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	task := model.TaskInput{
		ID:      input.ID,
		Date:    date,
		Title:   input.Title,
		Comment: input.Comment,
		Repeat:  input.Repeat,
	}

	err = db.Store.UpdateTask(&task)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			sendError(w, "task not found", http.StatusNotFound)
		} else {
			sendError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	sendJSON(w, map[string]any{})
}

// doneTaskHandler
func doneTaskHandler(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")

	if id == "" {
		http.Error(w, `{"error": "id not specified"}`, http.StatusBadRequest)
		return
	}

	task, err := db.Store.GetTask(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			sendError(w, "task not found", http.StatusNotFound)
		} else {
			sendError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	if task.Repeat == "" {

		err = db.Store.DeleteTask(id)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				sendError(w, "task not found", http.StatusNotFound)
			} else {
				sendError(w, "internal server error", http.StatusInternalServerError)
			}
			return
		}

		sendJSON(w, map[string]any{})
		return
	}

    now := time.Now()
    next, err := NextDate(now, task.Date, task.Repeat)
    if err != nil {
        sendError(w, err.Error(), http.StatusBadRequest)
        return
    }

    err = db.Store.UpdateTaskDate(task.ID, next)
    if err != nil {
        if strings.Contains(err.Error(), "not found") {
            sendError(w, "task not found", http.StatusNotFound)
        } else {
            sendError(w, "internal server error", http.StatusInternalServerError)
        }
        return
    }

    sendJSON(w, map[string]any{})
}

// deleteTaskHandler
func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")

	if id == "" {
		http.Error(w, `{"error": "id not specified"}`, http.StatusBadRequest)
		return
	}

	err := db.Store.DeleteTask(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			sendError(w, "task not found", http.StatusNotFound)
		} else {
			sendError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	sendJSON(w, map[string]any{})
}
