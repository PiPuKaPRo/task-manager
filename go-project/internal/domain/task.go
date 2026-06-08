package domain

import (
	"fmt"
	"time"
)

type TaskStatus string

const (
	StatusPending TaskStatus = "pending"
	StatusDone    TaskStatus = "done"
)

type TaskID int64

type Task struct {
	ID          TaskID     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	AssignedTo  *string    `json:"assigned_to,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	AssignedTo  string `json:"assigned_to"`
}

type UpdateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	AssignedTo  *string `json:"assigned_to"`
}

func (r *CreateTaskRequest) Validate() error {
	if r.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(r.Title) > 200 {
		return fmt.Errorf("title must be less than 200 characters")
	}
	return nil
}

func (r *UpdateTaskRequest) Validate() error {
	if r.Status != nil {
		if *r.Status != "pending" && *r.Status != "done" {
			return fmt.Errorf("status must be 'pending' or 'done'")
		}
	}
	return nil
}

func NewTask(title, description string, assignedTo *string) *Task {
	now := time.Now()
	return &Task{
		Title:       title,
		Description: description,
		Status:      StatusPending,
		AssignedTo:  assignedTo,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

var (
	ErrNotFound     = fmt.Errorf("task not found")
	ErrInvalidInput = fmt.Errorf("invalid input")
	ErrInternal     = fmt.Errorf("internal error")
)
