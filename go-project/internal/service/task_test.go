package service

import (
	"log/slog"
	"os"
	"testing"

	"task-manager/internal/domain"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

type mockRepo struct {
	tasks  map[domain.TaskID]*domain.Task
	nextID int64
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		tasks:  make(map[domain.TaskID]*domain.Task),
		nextID: 1,
	}
}

func (m *mockRepo) Create(task *domain.Task) error {
	task.ID = domain.TaskID(m.nextID)
	m.tasks[task.ID] = task
	m.nextID++
	return nil
}

func (m *mockRepo) GetByID(id domain.TaskID) (*domain.Task, error) {
	if task, ok := m.tasks[id]; ok {
		return task, nil
	}
	return nil, nil
}

func (m *mockRepo) Update(task *domain.Task) error {
	if _, ok := m.tasks[task.ID]; ok {
		m.tasks[task.ID] = task
		return nil
	}
	return domain.ErrNotFound
}

func (m *mockRepo) Delete(id domain.TaskID) error {
	delete(m.tasks, id)
	return nil
}

func (m *mockRepo) List(limit, offset int, status string) ([]*domain.Task, error) {
	var result []*domain.Task
	for _, task := range m.tasks {
		if status == "" || string(task.Status) == status {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *mockRepo) UpdateStatus(id domain.TaskID, status domain.TaskStatus) error {
	if task, ok := m.tasks[id]; ok {
		task.Status = status
		return nil
	}
	return domain.ErrNotFound
}

// ========== ТЕСТЫ ==========
func TestServiceCreate(t *testing.T) {
	repo := newMockRepo()
	svc := NewTaskService(repo, testLogger)

	req := &domain.CreateTaskRequest{Title: "Test Task"}
	task, err := svc.Create(nil, req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if task.Title != "Test Task" {
		t.Errorf("Expected 'Test Task', got '%s'", task.Title)
	}
}

func TestServiceCreateEmptyTitle(t *testing.T) {
	repo := newMockRepo()
	svc := NewTaskService(repo, testLogger)

	req := &domain.CreateTaskRequest{Title: ""}
	_, err := svc.Create(nil, req)
	if err == nil {
		t.Error("Expected error for empty title")
	}
}

func TestServiceGetByIDSuccess(t *testing.T) {
	repo := newMockRepo()
	svc := NewTaskService(repo, testLogger)

	req := &domain.CreateTaskRequest{Title: "Test"}
	task, _ := svc.Create(nil, req)

	found, err := svc.GetByID(nil, task.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if found.ID != task.ID {
		t.Errorf("Expected ID %d, got %d", task.ID, found.ID)
	}
}

func TestServiceGetByIDNotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewTaskService(repo, testLogger)

	_, err := svc.GetByID(nil, 999)
	if err != domain.ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestServiceUpdateSuccess(t *testing.T) {
	repo := newMockRepo()
	svc := NewTaskService(repo, testLogger)

	req := &domain.CreateTaskRequest{Title: "Original"}
	task, _ := svc.Create(nil, req)

	newTitle := "Updated"
	updateReq := &domain.UpdateTaskRequest{Title: &newTitle}
	err := svc.Update(nil, task.ID, updateReq)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	updated, _ := svc.GetByID(nil, task.ID)
	if updated.Title != newTitle {
		t.Errorf("Expected '%s', got '%s'", newTitle, updated.Title)
	}
}

func TestServiceDeleteSuccess(t *testing.T) {
	repo := newMockRepo()
	svc := NewTaskService(repo, testLogger)

	req := &domain.CreateTaskRequest{Title: "Test"}
	task, _ := svc.Create(nil, req)

	err := svc.Delete(nil, task.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = svc.GetByID(nil, task.ID)
	if err != domain.ErrNotFound {
		t.Error("Task should be deleted")
	}
}

func TestServiceList(t *testing.T) {
	repo := newMockRepo()
	svc := NewTaskService(repo, testLogger)

	svc.Create(nil, &domain.CreateTaskRequest{Title: "Task 1"})
	svc.Create(nil, &domain.CreateTaskRequest{Title: "Task 2"})

	tasks, err := svc.List(nil, 10, 0, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) < 2 {
		t.Errorf("Expected at least 2 tasks, got %d", len(tasks))
	}
}

func TestServiceMarkDoneSuccess(t *testing.T) {
	repo := newMockRepo()
	svc := NewTaskService(repo, testLogger)

	req := &domain.CreateTaskRequest{Title: "Test"}
	task, _ := svc.Create(nil, req)

	err := svc.MarkDone(nil, task.ID)
	if err != nil {
		t.Fatalf("MarkDone failed: %v", err)
	}

	task, _ = svc.GetByID(nil, task.ID)
	if task.Status != domain.StatusDone {
		t.Errorf("Expected done, got %s", task.Status)
	}
}

func TestServiceAssignSuccess(t *testing.T) {
	repo := newMockRepo()
	svc := NewTaskService(repo, testLogger)

	req := &domain.CreateTaskRequest{Title: "Test"}
	task, _ := svc.Create(nil, req)

	err := svc.Assign(nil, task.ID, "newuser@test.com")
	if err != nil {
		t.Fatalf("Assign failed: %v", err)
	}

	task, _ = svc.GetByID(nil, task.ID)
	if task.AssignedTo == nil || *task.AssignedTo != "newuser@test.com" {
		t.Error("Task not assigned correctly")
	}
}
