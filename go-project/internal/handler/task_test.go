package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"task-manager/internal/domain"
)

type MockTaskService struct {
	CreateFunc       func(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error)
	GetByIDFunc      func(ctx context.Context, id domain.TaskID) (*domain.Task, error)
	UpdateFunc       func(ctx context.Context, id domain.TaskID, req *domain.UpdateTaskRequest) error
	DeleteFunc       func(ctx context.Context, id domain.TaskID) error
	ListFunc         func(ctx context.Context, limit, offset int, status string) ([]*domain.Task, error)
	MarkDoneFunc     func(ctx context.Context, id domain.TaskID) error
	MarkUndoneFunc   func(ctx context.Context, id domain.TaskID) error
	AssignFunc       func(ctx context.Context, id domain.TaskID, assignedTo string) error
	ListByStatusFunc func(ctx context.Context, status string) ([]*domain.Task, error)
}

func (m *MockTaskService) Create(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return &domain.Task{ID: 1, Title: req.Title, Status: domain.StatusPending}, nil
}

func (m *MockTaskService) GetByID(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	if id == 999 {
		return nil, domain.ErrNotFound
	}
	return &domain.Task{ID: id, Title: "Test", Status: domain.StatusPending}, nil
}

func (m *MockTaskService) Update(ctx context.Context, id domain.TaskID, req *domain.UpdateTaskRequest) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	if id == 999 {
		return domain.ErrNotFound
	}
	return nil
}

func (m *MockTaskService) Delete(ctx context.Context, id domain.TaskID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	if id == 999 {
		return domain.ErrNotFound
	}
	return nil
}

func (m *MockTaskService) List(ctx context.Context, limit, offset int, status string) ([]*domain.Task, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, limit, offset, status)
	}
	return []*domain.Task{
		{ID: 1, Title: "Task 1", Status: domain.StatusPending},
		{ID: 2, Title: "Task 2", Status: domain.StatusDone},
	}, nil
}

func (m *MockTaskService) MarkDone(ctx context.Context, id domain.TaskID) error {
	if m.MarkDoneFunc != nil {
		return m.MarkDoneFunc(ctx, id)
	}
	if id == 999 {
		return domain.ErrNotFound
	}
	return nil
}

func (m *MockTaskService) MarkUndone(ctx context.Context, id domain.TaskID) error {
	if m.MarkUndoneFunc != nil {
		return m.MarkUndoneFunc(ctx, id)
	}
	if id == 999 {
		return domain.ErrNotFound
	}
	return nil
}

func (m *MockTaskService) Assign(ctx context.Context, id domain.TaskID, assignedTo string) error {
	if m.AssignFunc != nil {
		return m.AssignFunc(ctx, id, assignedTo)
	}
	if id == 999 {
		return domain.ErrNotFound
	}
	return nil
}

func (m *MockTaskService) ListByStatus(ctx context.Context, status string) ([]*domain.Task, error) {
	if m.ListByStatusFunc != nil {
		return m.ListByStatusFunc(ctx, status)
	}
	return []*domain.Task{{ID: 1, Title: "Task", Status: domain.StatusPending}}, nil
}

var testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

// ========== ТЕСТЫ ==========
func TestHealth(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestCreate_Success(t *testing.T) {
	mockService := &MockTaskService{
		CreateFunc: func(ctx context.Context, req *domain.CreateTaskRequest) (*domain.Task, error) {
			return &domain.Task{ID: 1, Title: req.Title, Status: domain.StatusPending}, nil
		},
	}
	handler := NewTaskHandler(mockService, testLogger)

	reqBody := `{"title":"Test Task"}`
	req := httptest.NewRequest("POST", "/tasks", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", w.Code)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestCreate_EmptyTitle(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	reqBody := `{"title":""}`
	req := httptest.NewRequest("POST", "/tasks", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestGet_Success(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	req := httptest.NewRequest("GET", "/tasks/1", nil)
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestGet_NotFound(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	req := httptest.NewRequest("GET", "/tasks/999", nil)
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestGet_InvalidID(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	req := httptest.NewRequest("GET", "/tasks/invalid", nil)
	w := httptest.NewRecorder()

	handler.Get(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdate_Success(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	reqBody := `{"title":"Updated"}`
	req := httptest.NewRequest("PUT", "/tasks/1", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	reqBody := `{"title":"Updated"}`
	req := httptest.NewRequest("PUT", "/tasks/999", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Update(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestDelete_Success(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	req := httptest.NewRequest("DELETE", "/tasks/1", nil)
	w := httptest.NewRecorder()

	handler.Delete(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d", w.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	req := httptest.NewRequest("DELETE", "/tasks/999", nil)
	w := httptest.NewRecorder()

	handler.Delete(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestList_Success(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	req := httptest.NewRequest("GET", "/tasks", nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestMarkDone_Success(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	req := httptest.NewRequest("POST", "/tasks/1/done", nil)
	w := httptest.NewRecorder()

	handler.MarkDone(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestMarkDone_NotFound(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	req := httptest.NewRequest("POST", "/tasks/999/done", nil)
	w := httptest.NewRecorder()

	handler.MarkDone(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestAssign_Success(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	reqBody := `{"assigned_to":"user@test.com"}`
	req := httptest.NewRequest("POST", "/tasks/1/assign", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Assign(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestListByStatus_Success(t *testing.T) {
	mockService := &MockTaskService{}
	handler := NewTaskHandler(mockService, testLogger)

	req := httptest.NewRequest("GET", "/tasks/status/pending", nil)
	w := httptest.NewRecorder()

	handler.ListByStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestResponseJSON(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, map[string]string{"test": "value"}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestResponseError(t *testing.T) {
	w := httptest.NewRecorder()
	Error(w, http.StatusBadRequest, "test error")

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Error != "test error" {
		t.Errorf("Expected 'test error', got '%s'", resp.Error)
	}
}
