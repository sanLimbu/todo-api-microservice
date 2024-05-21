package postgresql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sanLimbu/todo-api/internal"
)

//go:generate sqlc generate

func convertPriority(p Priority) (internal.Priority, error) {
	switch p {
	case PriorityNone:
		return internal.PriorityNone, nil
	case PriorityLow:
		return internal.PriorityLow, nil
	case PriorityMedium:
		return internal.PriorityMedium, nil
	case PriorityHigh:
		return internal.PriorityHigh, nil
	}

	return internal.Priority(-1), fmt.Errorf("unknown value: %s", p)
}

// newNullTime creates a sql.NullTime from a time.Time.
func newNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{
		Time:  t,
		Valid: !t.IsZero(),
	}
}

func newPriority(p internal.Priority) Priority {
	switch p {
	case internal.PriorityNone:
		return PriorityNone
	case internal.PriorityLow:
		return PriorityLow
	case internal.PriorityMedium:
		return PriorityMedium
	case internal.PriorityHigh:
		return PriorityHigh
	}
	return "invalid"
}
