package tasks

import (
	"github.com/Financial-Times/publish-carousel/native"

	"github.com/stretchr/testify/mock"
)

type MockTask struct {
	mock.Mock
}

func (m *MockTask) Prepare(collection string, uuid string) (*native.Content, string, error) {
	args := m.Called(collection, uuid)
	return args.Get(0).(*native.Content), args.String(1), args.Error(2)
}

func (m *MockTask) Execute(uuid string, content *native.Content, origin string, txId string) error {
	args := m.Called(uuid, content, origin, txId)
	return args.Error(0)
}
