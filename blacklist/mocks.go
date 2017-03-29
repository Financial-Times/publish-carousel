package blacklist

import (
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/stretchr/testify/mock"
)

type MockBlacklist struct {
	mock.Mock
}

func (m *MockBlacklist) ValidForPublish(uuid string, content *native.Content) (bool, error) {
	args := m.Called(uuid, content)
	return args.Bool(0), args.Error(1)
}
