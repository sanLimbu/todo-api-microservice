package rabbitmq

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"

	"github.com/sanLimbu/todo-api/internal"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

const otelName = "github.com/sanLimbu/todo-api/internal/rabbitmq"

//Task represents the repository used for publishing Task records.
type Task struct {
	ch *amqp.Channel
}

//NewTask instantiates the Task repository
func NewTask(channel *amqp.Channel) (*Task, error) {
	return &Task{
		ch: channel,
	}, nil
}

//Created publishes a message indicating a task was created.
func (t *Task) Created(ctx context.Context, task internal.Task) error {
	return t.publish(ctx, "Task.Created", "tasks.event.created", task)
}

//Deleted publishes a message indicating a task was deleted
func (t *Task) Deleted(ctx context.Context, id string) error {
	return t.publish(ctx, "Task.Deleted", "tasks.event.deleted", id)
}

//Updated publishes a message indicating a task was updated
func (t *Task) Updated(ctx context.Context, task internal.Task) error {
	return t.publish(ctx, "Task.Updated", "tasks.event.updated", task)
}

func (t *Task) publish(ctx context.Context, spanName, routingKey string, e interface{}) error {

	_, span := otel.Tracer(otelName).Start(ctx, spanName)

	//ctx, span := trace.SpanFromContext(ctx).Tracer().Start(ctx, spanName)
	defer span.End()

	span.SetAttributes(
		attribute.KeyValue{
			Key:   semconv.MessagingSystemKey,
			Value: attribute.StringValue("rabbitmq"),
		},
		attribute.KeyValue{
			Key:   semconv.MessagingRabbitmqRoutingKeyKey,
			Value: attribute.StringValue(routingKey),
		},
	)
	var b bytes.Buffer
	if err := gob.NewEncoder(&b).Encode(e); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeUnkown, "gob.Encode")
	}

	err := t.ch.Publish(
		"tasks",    //exchange
		routingKey, //routing key
		false,      //mandatory
		false,      //immediate
		amqp.Publishing{
			AppId:       "tasks-rest-server",
			ContentType: "application/x-encoding-gob",
			Body:        b.Bytes(),
			Timestamp:   time.Now(),
		})

	if err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeUnkown, "ch.publish")
	}
	return nil
}
