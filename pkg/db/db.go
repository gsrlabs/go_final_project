package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"

	"os"
	"todo/pkg/model"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// Storage represents database storage layer for scheduler tasks
type Storage struct {
	db *sql.DB
}

// TasksResp represents response structure for tasks list
type TasksResp struct {
	Tasks []*model.Task `json:"tasks"`
}

// Store is a global database storage instance
var Store *Storage

// Init initializes database connection and creates schema if needed
// dbFile - path to SQLite database file
func Init(dbFile string) error {
	_, err := os.Stat(dbFile)
	install := os.IsNotExist(err)

	conn, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Printf("ERROR: Failed to open database %s: %v", dbFile, err)
		return err
	}

	Store = &Storage{db: conn}

	if install {
		log.Printf("INFO: Database schema not found, creating new database")
		if _, err := Store.db.Exec(schema); err != nil {
			log.Printf("ERROR: Failed to create database schema: %v", err)
			return err
		}
		log.Printf("INFO: Database schema created successfully")
	}
	log.Printf("INFO: Database initialized successfully: %s", dbFile)
	return nil
}

// AddTask creates a new task in the scheduler
// Returns task ID or error
func (s *Storage) AddTask(task *model.Task) (int64, error) {
	log.Printf("DEBUG: Adding new task: %s", task.Title)

	result, err := s.db.Exec(`
		INSERT INTO scheduler (date, title, comment, repeat)
		VALUES (:date, :title, :comment, :repeat)
    `,
		sql.Named("date", task.Date),
		sql.Named("title", task.Title),
		sql.Named("comment", task.Comment),
		sql.Named("repeat", task.Repeat))

	if err != nil {
		log.Printf("ERROR: Database error in AddTask: %v", err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("ERROR: Failed to get last insert ID: %v", err)
		return 0, err
	}

	log.Printf("INFO: Task created successfully, ID: %d, Title: %s", id, task.Title)
	return id, nil
}

// GetTasks retrieves tasks list with pagination
// limit - maximum number of tasks to return
func (s *Storage) GetTasks(limit int) (TasksResp, error) {
	log.Printf("DEBUG: Getting tasks list, limit: %d", limit)

	rows, err := s.db.Query(`
        SELECT id, date, title, comment, repeat 
        FROM scheduler 
        ORDER BY date ASC 
        LIMIT :limit
    `, sql.Named("limit", limit))
	if err != nil {
		log.Printf("ERROR: Database error in GetTasks: %v", err)
		return TasksResp{}, err
	}
	defer rows.Close()

	var resp TasksResp
	resp.Tasks = make([]*model.Task, 0)

	for rows.Next() {
		t := &model.Task{}
		err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
		if err != nil {
			log.Printf("ERROR: Failed to scan task row: %v", err)
			return TasksResp{}, err
		}
		resp.Tasks = append(resp.Tasks, t)
	}

	if err = rows.Err(); err != nil {
		log.Printf("ERROR: Row iteration error in GetTasks: %v", err)
		return TasksResp{}, err
	}
	log.Printf("DEBUG: Retrieved %d tasks", len(resp.Tasks))
	return resp, nil
}

// GetTasksByTitle searches tasks by title or comment
// limit - maximum number of tasks to return
// search - search query string
func (s *Storage) GetTasksByTitle(limit int, search string) (TasksResp, error) {
	log.Printf("DEBUG: Searching tasks by title/comment: '%s', limit: %d", search, limit)

	rows, err := s.db.Query(`
        SELECT id, date, title, comment, repeat 
		FROM scheduler
        WHERE title LIKE :search OR comment LIKE :search
		ORDER BY date ASC
		LIMIT :limit
    `,
		sql.Named("search", "%"+search+"%"),
		sql.Named("limit", limit))
	if err != nil {
		log.Printf("ERROR: Database error in GetTasksByTitle: %v", err)
		return TasksResp{}, err

	}
	defer rows.Close()

	var resp TasksResp
	resp.Tasks = make([]*model.Task, 0)

	for rows.Next() {

		t := &model.Task{}
		err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
		if err != nil {
			log.Printf("ERROR: Failed to scan task row in GetTasksByTitle: %v", err)
			return TasksResp{}, err
		}
		resp.Tasks = append(resp.Tasks, t)
	}

	if err = rows.Err(); err != nil {
		log.Printf("ERROR: Row iteration error in GetTasksByTitle: %v", err)
		return TasksResp{}, err
	}
	log.Printf("DEBUG: Found %d tasks matching search: '%s'", len(resp.Tasks), search)
	return resp, nil

}

// GetTasksByDate retrieves tasks for specific date
// limit - maximum number of tasks to return
// date - target date in YYYYMMDD format
func (s *Storage) GetTasksByDate(limit int, date string) (TasksResp, error) {
	log.Printf("DEBUG: Getting tasks for date: %s, limit: %d", date, limit)
	rows, err := s.db.Query(`
        SELECT id, date, title, comment, repeat 
		FROM scheduler
        WHERE date = :date
		ORDER BY date ASC
		LIMIT :limit
    `,
		sql.Named("date", date),
		sql.Named("limit", limit))
	if err != nil {
		log.Printf("ERROR: Database error in GetTasksByDate: %v", err)
		return TasksResp{}, err
	}
	defer rows.Close()

	var resp TasksResp
	resp.Tasks = make([]*model.Task, 0)

	for rows.Next() {
		t := &model.Task{}
		err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
		if err != nil {
			log.Printf("ERROR: Failed to scan task row in GetTasksByDate: %v", err)
			return TasksResp{}, err
		}
		resp.Tasks = append(resp.Tasks, t)
	}

	if err = rows.Err(); err != nil {
		log.Printf("ERROR: Row iteration error in GetTasksByDate: %v", err)
		return TasksResp{}, err
	}
	log.Printf("DEBUG: Retrieved %d tasks for date: %s", len(resp.Tasks), date)
	return resp, nil

}

// GetTask retrieves single task by ID
// id - task identifier
func (s *Storage) GetTask(id string) (*model.Task, error) {
	log.Printf("DEBUG: Getting task by ID: %s", id)

	result := s.db.QueryRow(`
        SELECT id, date, title, comment, repeat 
		FROM scheduler
        WHERE id = :id
    `,
		sql.Named("id", id))

	var task model.Task
	err := result.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("WARN: Task not found, ID: %s", id)
		} else {
			log.Printf("ERROR: Database error in GetTask for ID %s: %v", id, err)
		}
		return nil, err
	}

	log.Printf("DEBUG: Task retrieved successfully, ID: %s", id)

	return &task, nil

}

// UpdateTask updates existing task
// task - task data with ID
func (s *Storage) UpdateTask(task *model.Task) error {
	log.Printf("DEBUG: Updating task, ID: %s", task.ID)

	resalt, err := s.db.Exec(`
        UPDATE scheduler
        SET date = :date,
            title = :title,
            comment = :comment,
            repeat = :repeat
        WHERE id = :id
    `,
		sql.Named("id", task.ID),
		sql.Named("date", task.Date),
		sql.Named("title", task.Title),
		sql.Named("comment", task.Comment),
		sql.Named("repeat", task.Repeat))

	if err != nil {
		log.Printf("ERROR: Database error in UpdateTask for ID %s: %v", task.ID, err)
		return err
	}

	count, err := resalt.RowsAffected()
	if err != nil {
		log.Printf("ERROR: Failed to get rows affected in UpdateTask: %v", err)
		return err
	}
	if count == 0 {
		log.Printf("WARN: Task not found for update, ID: %s", task.ID)
		return fmt.Errorf("task with id=%s not found (nothing updated)", task.ID)
	}
	log.Printf("INFO: Task updated successfully, ID: %s", task.ID)
	return nil
}

// UpdateTaskDate updates only task date
// id - task identifier
// date - new date in YYYYMMDD format
func (s *Storage) UpdateTaskDate(id, date string) error {
	log.Printf("DEBUG: Updating task date, ID: %s, new date: %s", id, date)

	resalt, err := s.db.Exec(`
        UPDATE scheduler
        SET date = :date
        WHERE id = :id
    `,
		sql.Named("date", date),
		sql.Named("id", id))

	if err != nil {
		log.Printf("ERROR: Database error in UpdateTaskDate for ID %s: %v", id, err)
		return err
	}

	count, err := resalt.RowsAffected()
	if err != nil {
		log.Printf("ERROR: Failed to get rows affected in UpdateTaskDate: %v", err)
		return err
	}
	if count == 0 {
		log.Printf("WARN: Task not found for date update, ID: %s", id)
		return fmt.Errorf("task with id=%s not found (nothing updated)", id)
	}
	log.Printf("INFO: Task date updated successfully, ID: %s, new date: %s", id, date)
	return nil
}

// DeleteTask removes task from scheduler
// id - task identifier
func (s *Storage) DeleteTask(id string) error {
	log.Printf("DEBUG: Deleting task, ID: %s", id)

	resalt, err := s.db.Exec(`
        DELETE FROM scheduler
        WHERE id = :id
    `,
		sql.Named("id", id))

	if err != nil {
		log.Printf("ERROR: Database error in DeleteTask for ID %s: %v", id, err)
		return err
	}

	count, err := resalt.RowsAffected()
	if err != nil {
		log.Printf("ERROR: Failed to get rows affected in DeleteTask: %v", err)
		return err
	}
	if count == 0 {
		log.Printf("WARN: Task not found for deletion, ID: %s", id)
		return fmt.Errorf("task with id=%s not found (nothing deleted)", id)
	}

	log.Printf("INFO: Task deleted successfully, ID: %s", id)
	return nil
}
