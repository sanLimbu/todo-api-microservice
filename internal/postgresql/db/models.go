package db

import (
	"database/sql"

	"github.com/google/uuid"
)

type Priority string

const (
	PriorityNone   Priority = "none"
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

type Tasks struct {
	ID          uuid.UUID
	Description string
	Priority    Priority
	StartDate   sql.NullTime
	DueDate     sql.NullTime
	Done        bool
}
