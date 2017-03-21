package scheduler

import "github.com/stretchr/testify/mock"

// MetadataRWMock is a mock of a MetadataReadWriter taht can be used to test
type MetadataRWMock struct {
	mock.Mock
}

func (m *MetadataRWMock) LoadMetadata(id string) (*CycleMetadata, error) {
	args := m.Called(id)
	return args.Get(0).(*CycleMetadata), args.Error(1)
}

func (m *MetadataRWMock) WriteMetadata(id string, state Cycle) error {
	args := m.Called(id, state)
	return args.Error(0)
}

// CycleMock is a mock of a Cycle taht can be used to test
type CycleMock struct {
	mock.Mock
}

func (m *CycleMock) ID() string {
	args := m.Called()
	return args.String(0)
}

func (m *CycleMock) Start() {
	m.Called()
}

func (m *CycleMock) Stop() {
	m.Called()
}

func (m *CycleMock) Reset() {
	m.Called()
}

func (m *CycleMock) Metadata() *CycleMetadata {
	args := m.Called()
	return args.Get(0).(*CycleMetadata)
}

func (m *CycleMock) RestoreMetadata(state *CycleMetadata) {
	m.Called(state)
}

func (m *CycleMock) TransformToConfig() *CycleConfig {
	args := m.Called()
	return args.Get(0).(*CycleConfig)
}
