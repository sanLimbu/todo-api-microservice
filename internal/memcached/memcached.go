package memcached

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/sanLimbu/todo-api/internal"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

const otelName = "github.com/sanLimbu/todo-api/internal/memcached"

func deleteTask(ctx context.Context, client *memcache.Client, key string) {

	defer newOTELSpan(ctx, "deleteTask").End()

	_ = client.Delete(key)

}

func getTask(ctx context.Context, client *memcache.Client, key string, target interface{}) error {

	defer newOTELSpan(ctx, "getTask").End()

	item, err := client.Get(key)
	if err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeUnkown, "client.Get")
	}

	if err := gob.NewDecoder(bytes.NewReader(item.Value)).Decode(target); err != nil {
		return internal.WrapErrorf(err, internal.ErrorCodeUnkown, "gob.NewDecoder")
	}

	return nil

}

func setTask(ctx context.Context, client *memcache.Client, key string, value interface{}, expiration time.Duration) {
	defer newOTELSpan(ctx, "setTask").End()

	var b bytes.Buffer

	if err := gob.NewEncoder(&b).Encode(value); err != nil {
		return
	}

	_ = client.Set(&memcache.Item{
		Key:        key,
		Value:      b.Bytes(),
		Expiration: int32(time.Now().Add(expiration).Unix()),
	})
}

func newOTELSpan(ctx context.Context, name string) trace.Span {
	_, span := otel.Tracer(otelName).Start(ctx, name)

	span.SetAttributes(semconv.DBSystemMemcached)

	return span
}
