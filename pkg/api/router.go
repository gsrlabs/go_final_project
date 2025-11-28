package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"todo/pkg/db"
	"todo/pkg/model"

	"github.com/go-chi/chi/v5"
)

const dateLayout = "20060102"
const limit int = 50

// Router sets up HTTP routes and middleware
// Returns configured HTTP handler
func Router() http.Handler {
	r := chi.NewRouter()

	r.Handle("/*", http.FileServer(http.Dir("./web")))

	// Public routes (no authentication required)
	r.Group(func(r chi.Router) {
		r.Get("/api/nextdate", nextDayHandler)
		r.Post("/api/signin", SignInHandler)
	})

	// Protected routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/api/task", addTaskHandler)
		r.Get("/api/task", getTaskHandler)
		r.Put("/api/task", updateTaskHandler)
		r.Get("/api/tasks", tasksHandler)
		r.Post("/api/task/done", doneTaskHandler)
		r.Delete("/api/task", deleteTaskHandler)
	})

	log.Printf("INFO: Router initialized with authentication middleware")
	return r
}

// authMiddleware - adapter for chi middleware
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AuthMiddleware(next.ServeHTTP)(w, r)
	})
}

// nextDayHandler calculates next date for recurring tasks
// GET /api/nextdate?now=YYYYMMDD&date=YYYYMMDD&repeat=rule
func nextDayHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Calculating next date for recurring task")

	nowStart := r.URL.Query().Get("now")
	dstart := r.URL.Query().Get("date")
	repeat := r.URL.Query().Get("repeat")

	if dstart == "" || repeat == "" {
		log.Printf("WARN: Missing required parameters for next date calculation")
		http.Error(w, `{"error":"date and repeat are required"}`, http.StatusBadRequest)
		return
	}

	var now time.Time
	if nowStart == "" {
		now = time.Now()
	} else {
		parsed, err := time.Parse(dateLayout, nowStart)
		if err != nil {
			log.Printf("WARN: Invalid now parameter format: %s", nowStart)
			http.Error(w, `{"error":"invalid now format"}`, http.StatusBadRequest)
			return
		}
		now = parsed
	}

	next, err := NextDate(now, dstart, repeat)
	if err != nil {
		log.Printf("WARN: Next date calculation failed: %v", err)
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	log.Printf("DEBUG: Next date calculated: %s -> %s with rule: %s", dstart, next, repeat)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(next))
}

// addTaskHandler creates a new task
// POST /api/task
func addTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Received task creation request")

	var input model.Task
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("WARN: Invalid JSON in task creation request: %v", err)
		sendError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if input.Title == "" {
		log.Printf("WARN: Empty title in task creation request")
		sendError(w, "the title is empty", http.StatusBadRequest)
		return
	}

	date, err := NormalizeDate(input.Date, input.Repeat)
	if err != nil {
		log.Printf("WARN: Date normalization failed: %v", err)
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
		log.Printf("ERROR: Failed to save task to database: %v", err)
		sendError(w, "saving error", http.StatusInternalServerError)
		return
	}

	log.Printf("INFO: Task created successfully, ID: %d, Title: %s", id, task.Title)
	sendJSON(w, map[string]any{"id": fmt.Sprintf("%d", id)})
}

// tasksHandler retrieves tasks list with optional search
// GET /api/tasks?search=query
func tasksHandler(w http.ResponseWriter, r *http.Request) {

	search := r.URL.Query().Get("search")
	log.Printf("DEBUG: Retrieving tasks, search: '%s'", search)

	var tasks db.TasksResp
	var err error

	if search == "" {
		tasks, err = db.Store.GetTasks(limit)
	} else {
		if date, err2 := time.Parse("02.01.2006", search); err2 == nil {
			y, m, d := date.Date()
			date := fmt.Sprintf("%d%02d%02d", y, m, d)
			log.Printf("DEBUG: Searching tasks by date: %s", date)
			tasks, err = db.Store.GetTasksByDate(limit, date)
		} else {
			log.Printf("DEBUG: Searching tasks by title/comment: '%s'", search)
			tasks, err = db.Store.GetTasksByTitle(limit, search)
		}
	}

	if err != nil {
		log.Printf("ERROR: Failed to retrieve tasks: %v", err)
		sendError(w, "failed to get tasks", http.StatusInternalServerError)
		return
	}

	log.Printf("INFO: Retrieved %d tasks for search: '%s'", len(tasks.Tasks), search)
	sendJSON(w, tasks)
}

// getTaskHandler retrieves single task by ID
// GET /api/task?id=task_id
func getTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
log.Printf("DEBUG: Retrieving task by ID: %s", id)

	if id == "" {
		log.Printf("WARN: Task ID not specified in request")
		http.Error(w, `{"error": "id not specified"}`, http.StatusBadRequest)
		return
	}

	task, err := db.Store.GetTask(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Printf("WARN: Task not found, ID: %s", id)
			sendError(w, "task not found", http.StatusNotFound)
		} else {
			log.Printf("ERROR: Database error retrieving task %s: %v", id, err)
			sendError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("INFO: Task retrieved successfully, ID: %s", id)
	sendJSON(w, task)
}

// updateTaskHandler updates existing task
// PUT /api/task
func updateTaskHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: Received task update request")

	var input model.Task
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("WARN: Invalid JSON in task update request: %v", err)
		sendError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if input.ID == "" {
		log.Printf("WARN: Missing task ID in update request")
		sendError(w, "id is required", http.StatusBadRequest)
		return
	}

	if input.Title == "" {
		log.Printf("WARN: Empty title in task update request, ID: %s", input.ID)
		sendError(w, "the title is empty", http.StatusBadRequest)
		return
	}

	date, err := NormalizeDate(input.Date, input.Repeat)
	if err != nil {
		log.Printf("WARN: Date normalization failed for task %s: %v", input.ID, err)
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	task := model.Task{
		ID:      input.ID,
		Date:    date,
		Title:   input.Title,
		Comment: input.Comment,
		Repeat:  input.Repeat,
	}

	err = db.Store.UpdateTask(&task)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Printf("WARN: Task not found for update, ID: %s", input.ID)
			sendError(w, "task not found", http.StatusNotFound)
		} else {
			log.Printf("ERROR: Database error updating task %s: %v", input.ID, err)
			sendError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("INFO: Task updated successfully, ID: %s, Title: %s", input.ID, input.Title)
	sendJSON(w, map[string]any{})
}

// doneTaskHandler marks task as done and handles recurrence
// POST /api/task/done?id=task_id
func doneTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	log.Printf("DEBUG: Marking task as done, ID: %s", id)

	if id == "" {
		log.Printf("WARN: Task ID not specified in done request")
		http.Error(w, `{"error": "id not specified"}`, http.StatusBadRequest)
		return
	}

	task, err := db.Store.GetTask(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Printf("WARN: Task not found for done operation, ID: %s", id)
			sendError(w, "task not found", http.StatusNotFound)
		} else {
			log.Printf("ERROR: Database error retrieving task %s for done: %v", id, err)
			sendError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	if task.Repeat == "" {
		// One-time task - delete it
		err = db.Store.DeleteTask(id)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Printf("WARN: Task not found for deletion, ID: %s", id)
				sendError(w, "task not found", http.StatusNotFound)
			} else {
				log.Printf("ERROR: Database error deleting task %s: %v", id, err)
				sendError(w, "internal server error", http.StatusInternalServerError)
			}
			return
		}
		log.Printf("INFO: One-time task completed and deleted, ID: %s", id)
		sendJSON(w, map[string]any{})
		return
	}

	// Recurring task - calculate next date
	now := time.Now()
	next, err := NextDate(now, task.Date, task.Repeat)
	if err != nil {
		log.Printf("WARN: Next date calculation failed for recurring task %s: %v", id, err)
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = db.Store.UpdateTaskDate(task.ID, next)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Printf("WARN: Task not found for date update, ID: %s", task.ID)
			sendError(w, "task not found", http.StatusNotFound)
		} else {
			log.Printf("ERROR: Database error updating task date %s: %v", task.ID, err)
			sendError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	log.Printf("INFO: Recurring task completed, ID: %s, next date: %s, rule: %s", task.ID, next, task.Repeat)
	sendJSON(w, map[string]any{})
}

// deleteTaskHandler removes task from scheduler
// DELETE /api/task?id=task_id
func deleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	log.Printf("DEBUG: Deleting task, ID: %s", id)

	if id == "" {
		log.Printf("WARN: Task ID not specified in delete request")
		http.Error(w, `{"error": "id not specified"}`, http.StatusBadRequest)
		return
	}

	err := db.Store.DeleteTask(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			log.Printf("WARN: Task not found for deletion, ID: %s", id)
			sendError(w, "task not found", http.StatusNotFound)
		} else {
			log.Printf("ERROR: Database error deleting task %s: %v", id, err)
			sendError(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("INFO: Task deleted successfully, ID: %s", id)
	sendJSON(w, map[string]any{})
}
