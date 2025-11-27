// pkg/db/db.go
package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	"os"
	"todo/pkg/model"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

type Storage struct {
	db *sql.DB
}

type TasksResp struct {
	Tasks []model.TaskInput `json:"tasks"`
}

func (s *Storage) GetStore() *Storage {
	return Store
}

var Store *Storage

// Init
func Init(dbFile string) error {
	_, err := os.Stat(dbFile)
	install := os.IsNotExist(err)

	conn, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return err
	}

	Store = &Storage{db: conn}

	if install {
		if _, err := Store.db.Exec(schema); err != nil {
			return err
		}
	}

	return nil
}

// AddTask
func (s *Storage) AddTask(task *model.Task) (int64, error) {

	resalt, err := s.db.Exec(`
		INSERT INTO scheduler (date, title, comment, repeat)
		VALUES (:date, :title, :comment, :repeat)
    `,
		sql.Named("date", task.Date),
		sql.Named("title", task.Title),
		sql.Named("comment", task.Comment),
		sql.Named("repeat", task.Repeat))

	if err != nil {
		return 0, err
	}
	return resalt.LastInsertId()
}

// GetTasks
func (s *Storage) GetTasks(limit int) (TasksResp, error) {
	rows, err := s.db.Query(`
        SELECT id, date, title, comment, repeat 
        FROM scheduler 
        ORDER BY date ASC 
        LIMIT :limit
    `, sql.Named("limit", limit))
	if err != nil {
		return TasksResp{}, err
	}
	defer rows.Close()

	var resp TasksResp
	resp.Tasks = []model.TaskInput{}

	for rows.Next() {
		var t model.TaskInput
		err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
		if err != nil {
			return TasksResp{}, err
		}
		resp.Tasks = append(resp.Tasks, t)
	}

	if err = rows.Err(); err != nil {
		return TasksResp{}, err
	}

	return resp, nil
}

// GetTasksByTitle
func (s *Storage) GetTasksByTitle(limit int, search string) (TasksResp, error) {

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
		return TasksResp{}, err

	}
	defer rows.Close()

	var resp TasksResp
	resp.Tasks = []model.TaskInput{}

	for rows.Next() {
		
		var t model.TaskInput
		err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
		if err != nil {
			return TasksResp{}, err
		}
		resp.Tasks = append(resp.Tasks, t)
	}

	if err = rows.Err(); err != nil {
		return TasksResp{}, err
	}

	return resp, nil

}

// GetTasksByDate
func (s *Storage) GetTasksByDate(limit int, date string) (TasksResp, error) {

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
		return TasksResp{}, err
	}
	defer rows.Close()

	var resp TasksResp
	resp.Tasks = []model.TaskInput{}

	for rows.Next() {
		var t model.TaskInput
		err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
		if err != nil {
			return TasksResp{}, err
		}
		resp.Tasks = append(resp.Tasks, t)
	}

	if err = rows.Err(); err != nil {
		return TasksResp{}, err
	}

	return resp, nil

}

// GetTaskById
func (s *Storage) GetTask(id string) (*model.TaskInput, error) {

	result := s.db.QueryRow(`
        SELECT id, date, title, comment, repeat 
		FROM scheduler
        WHERE id = :id
    `,
		sql.Named("id", id))

	var task model.TaskInput
	err := result.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		return &model.TaskInput{}, err
	}

	return &task, nil

}

// UpdateTask
func (s *Storage) UpdateTask(task *model.TaskInput) error {

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
		return err
	}

	count, err := resalt.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("task with id=%s not found (nothing updated)", task.ID)
	}
	return nil
}

// UpdateTaskDate
func (s *Storage) UpdateTaskDate(id, date string) error {

	resalt, err := s.db.Exec(`
        UPDATE scheduler
        SET date = :date
        WHERE id = :id
    `,
		sql.Named("date", date),
		sql.Named("id", id))
	
	if err != nil {
		return err
	}

	count, err := resalt.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("task with id=%s not found (nothing updated)", id)
	}

	return nil
}

// DeleteTask
func (s *Storage) DeleteTask(id string) error {

	resalt, err := s.db.Exec(`
        DELETE FROM scheduler
        WHERE id = :id
    `,
		sql.Named("id", id))

	if err != nil {
		return err
	}

	count, err := resalt.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("task with id=%s not found (nothing deleted)", id)
	}
	
	return nil
}