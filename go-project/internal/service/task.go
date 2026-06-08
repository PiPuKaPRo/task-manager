package service

import (
	"context"
	"log/slog"
	"time"

	"task-manager/internal/domain"
)

type TaskService interface {
	Create(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error)
	GetByID(ctx context.Context, id domain.TaskID) (*domain.Task, error)
	Update(ctx context.Context, id domain.TaskID, req *domain.UpdateTaskRequest) error
	Delete(ctx context.Context, id domain.TaskID) error
	List(ctx context.Context, limit, offset int, status string) ([]*domain.Task, error)
	MarkDone(ctx context.Context, id domain.TaskID) error
	MarkUndone(ctx context.Context, id domain.TaskID) error
	Assign(ctx context.Context, id domain.TaskID, assignedTo string) error
	ListByStatus(ctx context.Context, status string) ([]*domain.Task, error)
}

type TaskRepository interface {
	Create(task *domain.Task) error
	GetByID(id domain.TaskID) (*domain.Task, error)
	Update(task *domain.Task) error
	Delete(id domain.TaskID) error
	List(limit, offset int, status string) ([]*domain.Task, error)
	UpdateStatus(id domain.TaskID, status domain.TaskStatus) error
}

type taskService struct {
	repo   TaskRepository
	logger *slog.Logger
}

func NewTaskService(repo TaskRepository, logger *slog.Logger) TaskService {
	return &taskService{repo: repo, logger: logger}
}

func (s *taskService) Create(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error) {
	if req.Title == "" {
		return nil, domain.ErrInvalidInput
	}

	var assignedTo *string
	if req.AssignedTo != "" {
		assignedTo = &req.AssignedTo
	}

	task := domain.NewTask(req.Title, req.Description, assignedTo)

	if err := s.repo.Create(task); err != nil {
		s.logger.ErrorContext(ctx, "failed to create task", "error", err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "task created", "id", task.ID, "title", task.Title)
	return task, nil
}

func (s *taskService) GetByID(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
	task, err := s.repo.GetByID(id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get task", "id", id, "error", err)
		return nil, err
	}
	if task == nil {
		return nil, domain.ErrNotFound
	}
	return task, nil
}

func (s *taskService) Update(ctx context.Context, id domain.TaskID, req *domain.UpdateTaskRequest) error {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return domain.ErrNotFound
	}

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Status != nil {
		task.Status = domain.TaskStatus(*req.Status)
	}
	if req.AssignedTo != nil {
		task.AssignedTo = req.AssignedTo
	}
	task.UpdatedAt = time.Now()

	if err := s.repo.Update(task); err != nil {
		s.logger.ErrorContext(ctx, "failed to update task", "id", id, "error", err)
		return err
	}

	s.logger.InfoContext(ctx, "task updated", "id", id)
	return nil
}

func (s *taskService) Delete(ctx context.Context, id domain.TaskID) error {
	err := s.repo.Delete(id)
	if err != nil {
		if err.Error() == "task not found" {
			return domain.ErrNotFound
		}
		s.logger.ErrorContext(ctx, "failed to delete task", "id", id, "error", err)
		return err
	}
	s.logger.InfoContext(ctx, "task deleted", "id", id)
	return nil
}

func (s *taskService) List(ctx context.Context, limit, offset int, status string) ([]*domain.Task, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(limit, offset, status)
}

func (s *taskService) MarkDone(ctx context.Context, id domain.TaskID) error {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return domain.ErrNotFound
	}
	if task.Status == domain.StatusDone {
		return nil
	}
	return s.repo.UpdateStatus(id, domain.StatusDone)
}

func (s *taskService) MarkUndone(ctx context.Context, id domain.TaskID) error {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return domain.ErrNotFound
	}
	if task.Status == domain.StatusPending {
		return nil
	}
	return s.repo.UpdateStatus(id, domain.StatusPending)
}

func (s *taskService) Assign(ctx context.Context, id domain.TaskID, assignedTo string) error {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if task == nil {
		return domain.ErrNotFound
	}
	task.AssignedTo = &assignedTo
	task.UpdatedAt = time.Now()
	return s.repo.Update(task)
}

func (s *taskService) ListByStatus(ctx context.Context, status string) ([]*domain.Task, error) {
	if status != "pending" && status != "done" {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.List(1000, 0, status)
}
