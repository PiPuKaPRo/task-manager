package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"task-manager/internal/domain"
	"task-manager/internal/service"
)

type TaskHandler struct {
	svc    service.TaskService
	logger *slog.Logger
}

func NewTaskHandler(svc service.TaskService, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{svc: svc, logger: logger}
}

func (h *TaskHandler) Health(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var req domain.CreateTaskRequest
	if err := json.Unmarshal(body, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := req.Validate(); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	task, err := h.svc.Create(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create task", "error", err)
		if err == domain.ErrInvalidInput {
			Error(w, http.StatusBadRequest, err.Error())
			return
		}
		Error(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	JSON(w, http.StatusCreated, task, nil)
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			Error(w, http.StatusNotFound, "task not found")
			return
		}
		h.logger.Error("failed to get task", "id", id, "error", err)
		Error(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	JSON(w, http.StatusOK, task, nil)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var req domain.UpdateTaskRequest
	if err := json.Unmarshal(body, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := req.Validate(); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.Update(r.Context(), id, &req); err != nil {
		if err == domain.ErrNotFound {
			Error(w, http.StatusNotFound, "task not found")
			return
		}
		h.logger.Error("failed to update task", "id", id, "error", err)
		Error(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "task updated"}, nil)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if err == domain.ErrNotFound {
			Error(w, http.StatusNotFound, "task not found")
			return
		}
		h.logger.Error("failed to delete task", "id", id, "error", err)
		Error(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	JSON(w, http.StatusNoContent, nil, nil)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := h.parsePagination(r)
	status := r.URL.Query().Get("status")

	tasks, err := h.svc.List(r.Context(), limit, offset, status)
	if err != nil {
		h.logger.Error("failed to list tasks", "error", err)
		Error(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	JSON(w, http.StatusOK, tasks, map[string]int{
		"total":  len(tasks),
		"limit":  limit,
		"offset": offset,
	})
}

func (h *TaskHandler) MarkDone(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}

	if err := h.svc.MarkDone(r.Context(), id); err != nil {
		if err == domain.ErrNotFound {
			Error(w, http.StatusNotFound, "task not found")
			return
		}
		h.logger.Error("failed to mark task as done", "id", id, "error", err)
		Error(w, http.StatusInternalServerError, "failed to mark task as done")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "task marked as done"}, nil)
}

func (h *TaskHandler) MarkUndone(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}

	if err := h.svc.MarkUndone(r.Context(), id); err != nil {
		if err == domain.ErrNotFound {
			Error(w, http.StatusNotFound, "task not found")
			return
		}
		h.logger.Error("failed to mark task as undone", "id", id, "error", err)
		Error(w, http.StatusInternalServerError, "failed to mark task as undone")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "task marked as undone"}, nil)
}

func (h *TaskHandler) Assign(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid task id")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var req struct {
		AssignedTo string `json:"assigned_to"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.svc.Assign(r.Context(), id, req.AssignedTo); err != nil {
		if err == domain.ErrNotFound {
			Error(w, http.StatusNotFound, "task not found")
			return
		}
		h.logger.Error("failed to assign task", "id", id, "error", err)
		Error(w, http.StatusInternalServerError, "failed to assign task")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "task assigned"}, nil)
}

func (h *TaskHandler) ListByStatus(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimPrefix(r.URL.Path, "/tasks/status/")
	if status != "pending" && status != "done" {
		Error(w, http.StatusBadRequest, "invalid status")
		return
	}

	tasks, err := h.svc.ListByStatus(r.Context(), status)
	if err != nil {
		h.logger.Error("failed to list tasks by status", "status", status, "error", err)
		Error(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	JSON(w, http.StatusOK, tasks, map[string]int{"total": len(tasks)})
}

func (h *TaskHandler) parseID(r *http.Request) (domain.TaskID, error) {
	path := strings.TrimPrefix(r.URL.Path, "/tasks/")
	for _, suffix := range []string{"/done", "/undone", "/assign"} {
		path = strings.TrimSuffix(path, suffix)
	}
	id, err := strconv.ParseInt(path, 10, 64)
	return domain.TaskID(id), err
}

func (h *TaskHandler) parsePagination(r *http.Request) (limit, offset int) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit = 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset = 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	return limit, offset
}
