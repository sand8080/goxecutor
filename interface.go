package goxecutor

import "context"

// ExecutionStatus graph execution status
type ExecutionStatus string

// ExecutionPolicy defines graph execution policy
type ExecutionPolicy string

const (
	PolicyRevertOnError ExecutionPolicy = "REVERT_ON_ERROR"
	PolicyFailOnError   ExecutionPolicy = "FAIL_ON_ERROR"
	PolicyIgnoreError   ExecutionPolicy = "IGNORE_ERROR"
)

const (
	// StatusSuccess graph executed successfully
	StatusSuccess ExecutionStatus = "SUCCESS"
	// StatusError graph executed with error
	StatusError ExecutionStatus = "ERROR"
	// StatusCancelled graph execution cancelled
	StatusCancelled ExecutionStatus = "CANCELLED"
)

type TrnLogStorage interface {
}

type Task interface {
}

type Graph interface {
	Add(task *Task) error
	Check() error
	Exec(ctx context.Context, policy ExecutionPolicy, storage TrnLogStorage) (ExecutionStatus, error)
}
