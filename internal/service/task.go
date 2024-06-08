package service

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/mercari/go-circuitbreaker"
	"github.com/sanLimbu/todo-api/internal"
)

const otelName = "github.com/sanLimbu/todo-api/internal/service"

//TaskRepository defines the datasource handeling persisting Task Records

type TaskRepository interface {
	Create(ctx context.Context, args internal.CreateParams) (internal.Task, error)
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (internal.Task, error)
	Update(ctx context.Context, id string, description string, priority internal.Priority, dates internal.Dates, isDone bool) error
}

//TaskSearchRepository defines the datastore handling searching Task records
type TaskSearchRepository interface {
	Search(ctx context.Context, args internal.SearchParams) (internal.SearchResults, error)
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
	cb        *circuitbreaker.CircuitBreaker
}

//NewTask
func NewTask(logger *zap.Logger, repo TaskRepository, search TaskSearchRepository, msgBroker TaskMessageBrokerRepository) *Task {
	return &Task{
		repo:      repo,
		search:    search,
		msgBroker: msgBroker,
		cb: circuitbreaker.New(
			circuitbreaker.WithOpenTimeout(time.Minute*2),
			circuitbreaker.WithTripFunc(circuitbreaker.NewTripFuncConsecutiveFailures(3)),
			circuitbreaker.WithOnStateChangeHookFn(func(oldState, newState circuitbreaker.State) {
				logger.Info("state changed",
					zap.String("old", string(oldState)),
					zap.String("new", string(newState)),
				)
			}),
		),
	}
}

// By searches Tasks matching the received values.
func (t *Task) By(ctx context.Context, args internal.SearchParams) (_ internal.SearchResults, err error) {

	defer newOTELSpan(ctx, "Task.By").End()

	if !t.cb.Ready() {
		return internal.SearchResults{}, internal.NewErrorf(internal.ErrorCodeUnkown, "service not available")

	}

	defer func() {
		err = t.cb.Done(ctx, err)
	}()

	res, err := t.search.Search(ctx, args)
	if err != nil {
		return internal.SearchResults{}, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "Search")
	}

	return res, nil
}

//Create stores a new record
func (t *Task) Create(ctx context.Context, params internal.CreateParams) (internal.Task, error) {

	defer newOTELSpan(ctx, "Task.Create").End()

	if err := params.Validate(); err != nil {
		return internal.Task{}, internal.WrapErrorf(err, internal.ErrorCodeInvalidArgument, "params.Validate")
	}

	task, err := t.repo.Create(ctx, params)
	if err != nil {
		return internal.Task{}, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "repo.Created")
	}
	_ = t.msgBroker.Created(ctx, task)
	return task, nil
}

//Delete removes an existing Task from the datastore
func (t *Task) Delete(ctx context.Context, id string) error {

	defer newOTELSpan(ctx, "Task.Delete").End()

	if err := t.repo.Delete(ctx, id); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeUnkown, "Delete")
	}
	_ = t.msgBroker.Deleted(ctx, id)
	return nil
}

// Task gets an existing Task from the datastore.
func (t *Task) Task(ctx context.Context, id string) (internal.Task, error) {

	defer newOTELSpan(ctx, "Task.Task").End()

	task, err := t.repo.Find(ctx, id)
	if err != nil {
		return internal.Task{}, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "Find")
	}

	return task, nil
}

// Update updates an existing Task in the datastore.
func (t *Task) Update(ctx context.Context, id string, description string, priority internal.Priority, dates internal.Dates, isDone bool) error {

	defer newOTELSpan(ctx, "Task.Update").End()

	if err := t.repo.Update(ctx, id, description, priority, dates, isDone); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeUnkown, "repo.update")
	}
	{
		task, err := t.repo.Find(ctx, id)
		if err == nil {
			_ = t.msgBroker.Updated(ctx, task) // XXX: Ignoring errors on purpose
		}
	}

	return nil
}

func newOTELSpan(ctx context.Context, name string) trace.Span {
	_, span := otel.Tracer(otelName).Start(ctx, name)
	return span
}
