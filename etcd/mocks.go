package etcd

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockWatcher struct {
	mock.Mock
}

func (m *MockWatcher) Watch(ctx context.Context, key string, callback func(val string)) {
	m.Called(ctx, key, callback)
}
func (m *MockWatcher) Read(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}
