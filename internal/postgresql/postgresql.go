package postgresql

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sanLimbu/todo-api/internal"
	"github.com/sanLimbu/todo-api/internal/postgresql/db"
	"go.opentelemetry.io/otel"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

//go:generate sqlc generate

const otelName = "github.com/sanLimbu/todo-api/internal/postgresql"

func convertPriority(p db.Priority) (internal.Priority, error) {
	switch p {
	case db.PriorityNone:
		return internal.PriorityNone, nil
	case db.PriorityLow:
		return internal.PriorityLow, nil
	case db.PriorityMedium:
		return internal.PriorityMedium, nil
	case db.PriorityHigh:
		return internal.PriorityHigh, nil
	}

	return internal.Priority(-1), fmt.Errorf("unknown value: %s", p)
}

// newNullTime creates a sql.NullTime from a time.Time.
func newTimeStamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{
		Time:  t,
		Valid: !t.IsZero(),
	}
}

func newPriority(p internal.Priority) db.Priority {
	switch p {
	case internal.PriorityNone:
		return db.PriorityNone
	case internal.PriorityLow:
		return db.PriorityLow
	case internal.PriorityMedium:
		return db.PriorityMedium
	case internal.PriorityHigh:
		return db.PriorityHigh
	}
	return "invalid"
}

func newOTELSpan(ctx context.Context, name string) trace.Span {
	_, span := otel.Tracer(otelName).Start(ctx, name)

	span.SetAttributes(semconv.DBSystemPostgreSQL)

	return span
}
