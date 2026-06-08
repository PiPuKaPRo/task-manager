package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"task-manager/internal/config"
)

// ========== МОДЕЛИ ==========
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

// ========== ФОРМАТ ОТВЕТА ==========
type Response struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
	Meta  interface{} `json:"meta,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}, meta interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data, Meta: meta})
}

func respondError(w http.ResponseWriter, status int, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Error: errMsg})
}

// ========== РЕПОЗИТОРИЙ ==========
type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(task *Task) error {
	query := `INSERT INTO tasks (title, description, status, assigned_to, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	return r.db.QueryRow(query, task.Title, task.Description, task.Status, task.AssignedTo,
		task.CreatedAt, task.UpdatedAt).Scan(&task.ID)
}

func (r *TaskRepository) GetByID(id TaskID) (*Task, error) {
	query := `SELECT id, title, description, status, assigned_to, created_at, updated_at
	          FROM tasks WHERE id = $1`
	var task Task
	err := r.db.QueryRow(query, id).Scan(
		&task.ID, &task.Title, &task.Description, &task.Status,
		&task.AssignedTo, &task.CreatedAt, &task.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &task, err
}

func (r *TaskRepository) Update(task *Task) error {
	query := `UPDATE tasks SET title=$1, description=$2, status=$3, assigned_to=$4, updated_at=$5 WHERE id=$6`
	result, err := r.db.Exec(query, task.Title, task.Description, task.Status, task.AssignedTo, task.UpdatedAt, task.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

func (r *TaskRepository) Delete(id TaskID) error {
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

func (r *TaskRepository) List(limit, offset int, status string) ([]*Task, error) {
	var query string
	var rows *sql.Rows
	var err error
	if status != "" {
		query = `SELECT id, title, description, status, assigned_to, created_at, updated_at
		         FROM tasks WHERE status=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		rows, err = r.db.Query(query, status, limit, offset)
	} else {
		query = `SELECT id, title, description, status, assigned_to, created_at, updated_at
		         FROM tasks ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		rows, err = r.db.Query(query, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []*Task
	for rows.Next() {
		var task Task
		err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status,
			&task.AssignedTo, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (r *TaskRepository) UpdateStatus(id TaskID, status TaskStatus) error {
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

// ========== СЕРВИС ==========
type TaskService struct {
	repo   *TaskRepository
	logger *log.Logger
}

func NewTaskService(repo *TaskRepository, logger *log.Logger) *TaskService {
	return &TaskService{repo: repo, logger: logger}
}

func (s *TaskService) CreateTask(req *CreateTaskRequest) (*Task, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	now := time.Now()
	task := &Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.AssignedTo != "" {
		task.AssignedTo = &req.AssignedTo
	}
	if err := s.repo.Create(task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) GetTask(id TaskID) (*Task, error) {
	return s.repo.GetByID(id)
}

func (s *TaskService) UpdateTask(id TaskID, req *UpdateTaskRequest) error {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}
	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		task.Status = TaskStatus(*req.Status)
	}
	if req.AssignedTo != nil {
		task.AssignedTo = req.AssignedTo
	}
	task.UpdatedAt = time.Now()
	return s.repo.Update(task)
}

func (s *TaskService) DeleteTask(id TaskID) error {
	return s.repo.Delete(id)
}

func (s *TaskService) ListTasks(limit, offset int, status string) ([]*Task, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(limit, offset, status)
}

func (s *TaskService) MarkDone(id TaskID) error {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}
	if task.Status == StatusDone {
		return nil
	}
	return s.repo.UpdateStatus(id, StatusDone)
}

func (s *TaskService) MarkUndone(id TaskID) error {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}
	if task.Status == StatusPending {
		return nil
	}
	return s.repo.UpdateStatus(id, StatusPending)
}

func (s *TaskService) AssignTask(id TaskID, assignedTo string) error {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}
	task.AssignedTo = &assignedTo
	task.UpdatedAt = time.Now()
	return s.repo.Update(task)
}

// ========== HTTP ХЕНДЛЕР ==========
type TaskHandler struct {
	service *TaskService
	logger  *log.Logger
}

func NewTaskHandler(service *TaskService, logger *log.Logger) *TaskHandler {
	return &TaskHandler{service: service, logger: logger}
}

func (h *TaskHandler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	task, err := h.service.CreateTask(&req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, task, nil)
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		respondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}
	idStr := parts[2]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}
	task, err := h.service.GetTask(TaskID(id))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if task == nil {
		respondError(w, http.StatusNotFound, "Task not found")
		return
	}
	respondJSON(w, http.StatusOK, task, nil)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		respondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}
	idStr := parts[2]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}
	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.service.UpdateTask(TaskID(id), &req); err != nil {
		if err.Error() == "task not found" {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Task updated"}, nil)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		respondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}
	idStr := parts[2]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}
	if err := h.service.DeleteTask(TaskID(id)); err != nil {
		if err.Error() == "task not found" {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusNoContent, nil, nil)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 100
	offset := 0
	status := r.URL.Query().Get("status")

	tasks, err := h.service.ListTasks(limit, offset, status)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, tasks, map[string]int{"total": len(tasks), "limit": limit, "offset": offset})
}

func (h *TaskHandler) MarkDone(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		respondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}
	idStr := parts[2]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}
	if err := h.service.MarkDone(TaskID(id)); err != nil {
		if err.Error() == "task not found" {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Task marked as done"}, nil)
}

func (h *TaskHandler) MarkUndone(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		respondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}
	idStr := parts[2]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}
	if err := h.service.MarkUndone(TaskID(id)); err != nil {
		if err.Error() == "task not found" {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Task marked as undone"}, nil)
}

func (h *TaskHandler) Assign(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		respondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}
	idStr := parts[2]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}
	var req struct {
		AssignedTo string `json:"assigned_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := h.service.AssignTask(TaskID(id), req.AssignedTo); err != nil {
		if err.Error() == "task not found" {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Task assigned"}, nil)
}

func (h *TaskHandler) ListByStatus(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimPrefix(r.URL.Path, "/tasks/status/")
	if status != "pending" && status != "done" {
		respondError(w, http.StatusBadRequest, "Invalid status")
		return
	}
	tasks, err := h.service.ListTasks(1000, 0, status)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, tasks, map[string]int{"total": len(tasks)})
}

// ========== MAIN ==========
func main() {
	// Загружаем конфигурацию из .env
	cfg := config.Load()

	logger := log.New(os.Stdout, "[APP] ", log.LstdFlags)
	logger.Println("Starting Task Manager API...")
	logger.Printf("Config loaded: host=%s port=%s user=%s dbname=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBName)

	// Формируем строку подключения из конфига
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		logger.Printf("Warning: Could not connect to PostgreSQL: %v", err)
		logger.Println("Starting without database - some features may not work")
		db = nil
	} else {
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(25)
		db.SetConnMaxLifetime(5 * time.Minute)

		if err := db.Ping(); err != nil {
			logger.Printf("Warning: Database ping failed: %v", err)
			db = nil
		} else {
			logger.Println("Database connected successfully")

			createTableSQL := `
			CREATE TABLE IF NOT EXISTS tasks (
				id BIGSERIAL PRIMARY KEY,
				title VARCHAR(200) NOT NULL,
				description TEXT,
				status VARCHAR(20) NOT NULL DEFAULT 'pending',
				assigned_to VARCHAR(100),
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);
			CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
			CREATE INDEX IF NOT EXISTS idx_tasks_assigned_to ON tasks(assigned_to);
			`
			if _, err := db.Exec(createTableSQL); err != nil {
				logger.Printf("Warning: Failed to create table: %v", err)
			} else {
				logger.Println("Table 'tasks' created/verified successfully")
			}
		}
	}

	repo := NewTaskRepository(db)
	service := NewTaskService(repo, logger)
	handler := NewTaskHandler(service, logger)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /tasks", handler.Create)
	mux.HandleFunc("GET /tasks", handler.List)
	mux.HandleFunc("GET /tasks/", handler.Get)
	mux.HandleFunc("PUT /tasks/", handler.Update)
	mux.HandleFunc("DELETE /tasks/", handler.Delete)
	mux.HandleFunc("POST /tasks/{id}/done", handler.MarkDone)
	mux.HandleFunc("POST /tasks/{id}/undone", handler.MarkUndone)
	mux.HandleFunc("POST /tasks/{id}/assign", handler.Assign)
	mux.HandleFunc("GET /health", handler.Health)
	mux.HandleFunc("GET /tasks/status/", handler.ListByStatus)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: mux,
	}

	go func() {
		logger.Println("========================================")
		logger.Println("Task Manager API is running!")
		logger.Println("========================================")
		logger.Printf("Server: http://localhost:%s", cfg.ServerPort)
		logger.Println("")
		logger.Println("Available endpoints:")
		logger.Println("  GET    /health                - Health check")
		logger.Println("  POST   /tasks                 - Create task")
		logger.Println("  GET    /tasks                 - List tasks")
		logger.Println("  GET    /tasks/{id}            - Get task")
		logger.Println("  PUT    /tasks/{id}            - Update task")
		logger.Println("  DELETE /tasks/{id}            - Delete task")
		logger.Println("  POST   /tasks/{id}/done       - Mark as done")
		logger.Println("  POST   /tasks/{id}/undone     - Mark as undone")
		logger.Println("  POST   /tasks/{id}/assign     - Assign task")
		logger.Println("  GET    /tasks/status/{status} - Filter by status")
		logger.Println("========================================")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Printf("Server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Println("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Printf("Shutdown error: %v", err)
	}

	logger.Println("Server stopped")
}
