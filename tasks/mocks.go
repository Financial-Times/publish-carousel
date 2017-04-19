package tasks

import "github.com/stretchr/testify/mock"

type MockTask struct {
	mock.Mock
}

func (m *MockTask) Publish(origin string, collection string, uuid string) error {
	args := m.Called(origin, collection, uuid)
	return args.Error(0)
}
