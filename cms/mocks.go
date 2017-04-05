package cms

import (
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/stretchr/testify/mock"
)

type MockNotifier struct {
	mock.Mock
}

func (m *MockNotifier) Notify(origin string, tid string, content *native.Content, hash string) error {
	args := m.Called(origin, tid, content, hash)
	return args.Error(0)
}

func (m *MockNotifier) GTG() error {
	args := m.Called()
	return args.Error(0)
}
