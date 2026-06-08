package repository

import (
	"database/sql"
	"fmt"

	"task-manager/internal/domain"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(task *domain.Task) error {
	query := `
		INSERT INTO tasks (title, description, status, assigned_to, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	return r.db.QueryRow(query,
		task.Title, task.Description, task.Status, task.AssignedTo,
		task.CreatedAt, task.UpdatedAt,
	).Scan(&task.ID)
}

func (r *TaskRepository) GetByID(id domain.TaskID) (*domain.Task, error) {
	query := `
		SELECT id, title, description, status, assigned_to, created_at, updated_at
		FROM tasks WHERE id = $1
	`
	var task domain.Task
	err := r.db.QueryRow(query, id).Scan(
		&task.ID, &task.Title, &task.Description, &task.Status,
		&task.AssignedTo, &task.CreatedAt, &task.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &task, err
}

func (r *TaskRepository) Update(task *domain.Task) error {
	query := `
		UPDATE tasks 
		SET title=$1, description=$2, status=$3, assigned_to=$4, updated_at=$5
		WHERE id=$6
	`
	result, err := r.db.Exec(query,
		task.Title, task.Description, task.Status, task.AssignedTo,
		task.UpdatedAt, task.ID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

func (r *TaskRepository) Delete(id domain.TaskID) error {
	query := `DELETE FROM tasks WHERE id=$1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

func (r *TaskRepository) List(limit, offset int, status string) ([]*domain.Task, error) {
	var query string
	var rows *sql.Rows
	var err error

	if status != "" {
		query = `
			SELECT id, title, description, status, assigned_to, created_at, updated_at
			FROM tasks WHERE status=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
		`
		rows, err = r.db.Query(query, status, limit, offset)
	} else {
		query = `
			SELECT id, title, description, status, assigned_to, created_at, updated_at
			FROM tasks ORDER BY created_at DESC LIMIT $1 OFFSET $2
		`
		rows, err = r.db.Query(query, limit, offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		var task domain.Task
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status,
			&task.AssignedTo, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (r *TaskRepository) UpdateStatus(id domain.TaskID, status domain.TaskStatus) error {
	query := `UPDATE tasks SET status=$1, updated_at=NOW() WHERE id=$2`
	result, err := r.db.Exec(query, status, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}
