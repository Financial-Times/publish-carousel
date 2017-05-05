package cluster

import "github.com/stretchr/testify/mock"

type MockService struct {
	mock.Mock
}

func (m *MockService) Check() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockService) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockService) ServiceName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockService) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockService) Description() string {
	args := m.Called()
	return args.String(0)
}
