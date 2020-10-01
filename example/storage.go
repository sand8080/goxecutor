package example

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/sand8080/goxecutor/task"
)

// MockedStorage mocks storage
type MockedStorage struct {
	mock.Mock
}

// NewMockedStorage creates MockedStorage
func NewMockedStorage() *MockedStorage {
	storage := MockedStorage{}
	storage.On("Save", mock.Anything).Return(nil)
	return &storage
}

// Save mocks task saving to storage
func (m *MockedStorage) Save(task *task.Task) error {
	args := m.Called(task)
	return args.Error(0)
}

// Load mocks task loading from storage
func (m *MockedStorage) Load(uuid uuid.UUID) (*task.Task, error) {
	args := m.Called(uuid)
	return args.Get(0).(*task.Task), args.Error(1)
}
