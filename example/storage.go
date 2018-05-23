package example

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/mock"

	"github.com/sand8080/goxecutor/task"
)

type MockedStorage struct {
	mock.Mock
}

func NewMockedStorage() *MockedStorage {
	storage := MockedStorage{}
	storage.On("Save", mock.Anything).Return(nil)
	return &storage
}

func (m *MockedStorage) Save(task *task.Task) error {
	args := m.Called(task)
	return args.Error(0)
}

func (m *MockedStorage) Load(uuid uuid.UUID) (*task.Task, error) {
	args := m.Called(uuid)
	return args.Get(0).(*task.Task), args.Error(1)
}
