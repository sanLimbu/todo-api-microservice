package memcached

import (
	"context"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/sanLimbu/todo-api/internal"
	"go.uber.org/zap"
)

type Task struct {
	client     *memcache.Client
	orig       TaskStore
	expiration time.Duration
	logger     *zap.Logger
}

type TaskStore interface {
	Create(ctx context.Context, params internal.CreateParams) (internal.Task, error)
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (internal.Task, error)
	Update(ctx context.Context, id string, description string, priority internal.Priority, dates internal.Dates, isDone bool) error
}

func NewTask(client *memcache.Client, orig TaskStore, logger *zap.Logger) *Task {
	return &Task{
		client:     client,
		orig:       orig,
		expiration: 15 * time.Minute,
		logger:     logger,
	}
}

func (t *Task) Create(ctx context.Context, params internal.CreateParams) (internal.Task, error) {
	defer newOTELSpan(ctx, "Task.Create").End()

	task, err := t.orig.Create(ctx, params)
	if err != nil {
		return internal.Task{}, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "orig.Create")
	}

	t.logger.Info("Create: setting value")

	setTask(ctx, t.client, task.ID, &task, t.expiration)
	return task, nil
}

func (t *Task) Delete(ctx context.Context, id string) error {
	defer newOTELSpan(ctx, "Task.Delete").End()

	if err := t.orig.Delete(ctx, id); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeUnkown, "orig.Delete")
	}
	deleteTask(ctx, t.client, id)
	return nil
}

func (t *Task) Find(ctx context.Context, id string) (internal.Task, error) {
	defer newOTELSpan(ctx, "Task.Find").End()

	var res internal.Task

	if err := getTask(ctx, t.client, id, &res); err != nil {
		return res, nil
	}

	t.logger.Info("Find: not found, let's cache it")

	// Cache-Aside Caching

	res, err := t.orig.Find(ctx, id)
	if err != nil {
		return res, internal.WrapErrorf(err, internal.ErrorCodeUnkown, "orig.Find")
	}

	setTask(ctx, t.client, res.ID, &res, t.expiration)

	return res, nil
}

func (t *Task) Update(ctx context.Context, id string, description string, priority internal.Priority, dates internal.Dates, isDone bool) error {
	defer newOTELSpan(ctx, "Task.Update").End()

	if err := t.orig.Update(ctx, id, description, priority, dates, isDone); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeUnkown, "orig.Update")

	}

	t.logger.Info("Update: setting value")

	deleteTask(ctx, t.client, id)

	task, err := t.orig.Find(ctx, id)
	if err != nil {
		return nil
	}

	setTask(ctx, t.client, task.ID, &task, t.expiration)
	return nil
}
