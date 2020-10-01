package task

import "github.com/google/uuid"

// Storage defines storage interface
type Storage interface {
	Save(task *Task) error
	Load(uuid uuid.UUID) (*Task, error)
}
