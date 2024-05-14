package service

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"

	"github.com/sanLimbu/todo-api/internal"
)

//TaskRepository defines the datasource handeling persisting Task Records

type TaskRepository interface {
	Create(ctx context.Context, description string, priority internal.Priority, dates internal.Dates) (internal.Task, error)
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (internal.Task, error)
	Update(ctx context.Context, id string, description string, priority internal.Priority, dates internal.Dates, isDone bool) error
}

//TaskSearchRepository defines the datastore handling searching Task records
type TaskSearchRepository interface {
	Search(ctx context.Context, description *string, priority *internal.Priority, isDone *bool) ([]internal.Task, error)
}

//TaskMessageBrokerRepository defines the datasource handling persisting Searchable Task Records
type TaskMessageBrokerRepository interface {
	Created(ctx context.Context, task internal.Task) error
	Deleted(ctx context.Context, id string) error
	Updated(ctx context.Context, task internal.Task) error
}

//Task defines the application service in charge of interacting with Tasks
type Task struct {
	repo      TaskRepository
	search    TaskSearchRepository
	msgBroker TaskMessageBrokerRepository
}

//NewTask
func NewTask(repo TaskRepository, search TaskSearchRepository, msgBroker TaskMessageBrokerRepository) *Task {
	return &Task{
		repo:      repo,
		search:    search,
		msgBroker: msgBroker,
	}
}

// By searches Tasks matching the received values.
func (t *Task) By(ctx context.Context, description *string, priority *internal.Priority, isDone *bool) ([]internal.Task, error) {
	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.By")
	defer span.End()

	res, err := t.search.Search(ctx, description, priority, isDone)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return res, nil
}

//Create stores a new record
func (t *Task) Create(ctx context.Context, description string, priority internal.Priority, dates internal.Dates) (internal.Task, error) {

	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.Create")
	defer span.End()

	task, err := t.repo.Create(ctx, description, priority, dates)
	if err != nil {
		return internal.Task{}, fmt.Errorf("repo created: %w", err)
	}
	_ = t.msgBroker.Created(ctx, task)
	return task, nil
}

//Delete removes an existing Task from the datastore
func (t *Task) Delete(ctx context.Context, id string) error {
	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.Delete")
	defer span.End()

	if err := t.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("repo deleted: %w", err)
	}
	_ = t.msgBroker.Deleted(ctx, id)
	return nil
}

// Task gets an existing Task from the datastore.
func (t *Task) Task(ctx context.Context, id string) (internal.Task, error) {
	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.Task")
	defer span.End()

	task, err := t.repo.Find(ctx, id)
	if err != nil {
		return internal.Task{}, fmt.Errorf("repo find: %w", err)
	}

	return task, nil
}

// Update updates an existing Task in the datastore.
func (t *Task) Update(ctx context.Context, id string, description string, priority internal.Priority, dates internal.Dates, isDone bool) error {
	ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, "Task.Update")
	defer span.End()

	if err := t.repo.Update(ctx, id, description, priority, dates, isDone); err != nil {
		return fmt.Errorf("repo update: %w", err)
	}

	{
		// XXX: This will be improved when Kafka events are introduced in future episodes
		task, err := t.repo.Find(ctx, id)
		if err == nil {
			_ = t.msgBroker.Updated(ctx, task) // XXX: Ignoring errors on purpose
		}
	}

	return nil
}
