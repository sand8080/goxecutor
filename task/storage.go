package task

import "github.com/satori/go.uuid"

type Storage interface {
	Save(task *Task) error
	Load(uuid uuid.UUID) (*Task, error)
}
