package domain

import (
	"testing"
)

func TestCreateTaskRequestValidate(t *testing.T) {
	req := &CreateTaskRequest{Title: "Valid Title"}
	if err := req.Validate(); err != nil {
		t.Errorf("Valid request should not error: %v", err)
	}

	req.Title = ""
	if err := req.Validate(); err == nil {
		t.Error("Empty title should return error")
	}
}

func TestUpdateTaskRequestValidate(t *testing.T) {
	status := "done"
	req := &UpdateTaskRequest{Status: &status}
	if err := req.Validate(); err != nil {
		t.Errorf("Valid status should not error: %v", err)
	}

	invalidStatus := "invalid"
	req.Status = &invalidStatus
	if err := req.Validate(); err == nil {
		t.Error("Invalid status should return error")
	}
}

func TestNewTask(t *testing.T) {
	assignedTo := "user@test.com"
	task := NewTask("Test Title", "Test Desc", &assignedTo)

	if task.Title != "Test Title" {
		t.Errorf("Expected 'Test Title', got '%s'", task.Title)
	}
	if task.Status != StatusPending {
		t.Errorf("Expected pending, got '%s'", task.Status)
	}
	if task.AssignedTo == nil || *task.AssignedTo != "user@test.com" {
		t.Error("AssignedTo not set correctly")
	}
}

func TestConstants(t *testing.T) {
	if StatusPending != "pending" {
		t.Error("StatusPending should be 'pending'")
	}
	if StatusDone != "done" {
		t.Error("StatusDone should be 'done'")
	}
}
