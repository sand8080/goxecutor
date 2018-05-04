package task

import (
	"context"
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/mock"
)

type mockedStorage struct {
	mock.Mock
}

func newMockedStorage() *mockedStorage {
	storage := mockedStorage{}
	storage.On("Save", mock.Anything).Return(nil)
	return &storage
}

func (m *mockedStorage) Save(task *Task) error {
	args := m.Called(task)
	return args.Error(0)
}

func (m *mockedStorage) Load(uuid uuid.UUID) (*Task, error) {
	args := m.Called(uuid)
	return args.Get(0).(*Task), args.Error(1)
}

func Test_SavedToStorage(t *testing.T) {
	task := NewTask("T", nil, nil, dumpDoFunc, nil)
	storage := new(mockedStorage)
	storage.On("Save", task).Return(nil)

	ctx, cancelFunc := context.WithCancel(context.Background())
	Do(ctx, cancelFunc, task, storage)

	storage.AssertExpectations(t)
}
