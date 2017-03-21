package tasks

import "github.com/stretchr/testify/mock"

type PublishTaskMock struct {
	mock.Mock
}

func (m *PublishTaskMock) Publish(origin string, collection string, uuid string) error {
	args := m.Called(origin, collection, uuid)
	return args.Error(0)
}
