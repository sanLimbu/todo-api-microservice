package rest

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sanLimbu/todo-api/internal"
)

const uuidRegEx string = `[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}`

//go:generate counterfeiter -o resttesting/task_service.gen.go . TaskService

//TaskService ...
type TaskService interface {
	By(ctx context.Context, description *string, priority *internal.Priority, isDone *bool) ([]internal.Task, error)
	Create(ctx context.Context, description string, priority internal.Priority, dates internal.Dates) (internal.Task, error)
	Delete(ctx context.Context, id string) error
	Task(ctx context.Context, id string) (internal.Task, error)
	Update(ctx context.Context, id string, description string, priority internal.Priority, dates internal.Dates, isDone bool) error
}

//TaskHandler ...
type TaskHandler struct {
	svc TaskService
}

//NewTaskHandler
func NewTaskHandler(svc TaskService) *TaskHandler {
	return &TaskHandler{
		svc: svc,
	}
}

//Register connects the handlers to the router
func (t *TaskHandler) Register(r *mux.Router) {
	//r.HandleFunc("/tasks", t.crea)
}

// Task is an activity that needs to be completed within a period of time.
type Task struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Priority    Priority `json:"priority"`
	Dates       Dates    `json:"dates"`
	IsDone      bool     `json:"is_done"`
}

// CreateTasksRequest defines the request used for creating tasks.
type CreateTasksRequest struct {
	Description string   `json:"description"`
	Priority    Priority `json:"priority"`
	Dates       Dates    `json:"dates"`
}

// CreateTasksResponse defines the response returned back after creating tasks.
type CreateTasksResponse struct {
	Task Task `json:"task"`
}

func (t *TaskHandler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateTasksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		renderErrorResponse(r.Context(), w, "invalid request", internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "json decoder"))
		return
	}
	defer r.Body.Close()

	task, err := t.svc.Create(r.Context(), req.Description, req.Priority.Convert(), req.Dates.Convert())
	if err != nil {
		renderErrorResponse(r.Context(), w, "create failed", err)
		return
	}
	renderResponse(w, &CreateTasksResponse{
		Task: Task{
			ID:          task.ID,
			Description: task.Description,
			Priority:    NewPriority(task.Priority),
			Dates:       NewDates(task.Dates),
		},
	},
		http.StatusCreated)
}

func (t *TaskHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, _ := mux.Vars(r)["id"] // NOTE: Safe to ignore error, because it's always defined.
	if err := t.svc.Delete(r.Context(), id); err != nil {
		renderErrorResponse(r.Context(), w, "delete failed", err)
		return
	}
	renderResponse(w, struct{}{}, http.StatusOK)
}

//ReadTaskResponse defines the response returned back after searching one task
type ReadTaskResponse struct {
	Task Task `json:"task"`
}
