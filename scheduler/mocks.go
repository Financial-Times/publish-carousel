package scheduler

import (
	"time"

	"github.com/stretchr/testify/mock"
)

// MetadataRWMock is a mock of a MetadataReadWriter taht can be used to test
type MockMetadataRW struct {
	mock.Mock
}

func (m *MockMetadataRW) LoadMetadata(id string) (CycleMetadata, error) {
	args := m.Called(id)
	return args.Get(0).(CycleMetadata), args.Error(1)
}

func (m *MockMetadataRW) WriteMetadata(id string, state Cycle) error {
	args := m.Called(id, state)
	return args.Error(0)
}

type MockScheduler struct {
	mock.Mock
}

func (m *MockScheduler) Cycles() map[string]Cycle {
	args := m.Called()
	return args.Get(0).(map[string]Cycle)
}
func (m *MockScheduler) Throttles() map[string]Throttle {
	args := m.Called()
	return args.Get(0).(map[string]Throttle)
}

func (m *MockScheduler) AddThrottle(name string, throttleInterval string) error {
	args := m.Called(name, throttleInterval)
	return args.Error(0)
}

func (m *MockScheduler) DeleteThrottle(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockScheduler) NewCycle(config CycleConfig) (Cycle, error) {
	args := m.Called(config)
	return args.Get(0).(Cycle), args.Error(1)
}

func (m *MockScheduler) AddCycle(cycle Cycle) error {
	args := m.Called(cycle)
	return args.Error(0)
}

func (m *MockScheduler) DeleteCycle(cycleID string) error {
	args := m.Called(cycleID)
	return args.Error(0)
}

func (m *MockScheduler) RestorePreviousState() {
	m.Called()
}

func (m *MockScheduler) SaveCycleMetadata() {
	m.Called()
}

func (m *MockScheduler) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockScheduler) Shutdown() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockScheduler) ToggleHandler(toggleValue string) {
	m.Called(toggleValue)
}

type MockCycle struct {
	mock.Mock
}

func (m *MockCycle) ID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockCycle) Start() {
	m.Called()
}

func (m *MockCycle) Stop() {
	m.Called()
}

func (m *MockCycle) Reset() {
	m.Called()
}

func (m *MockCycle) Metadata() CycleMetadata {
	args := m.Called()
	return args.Get(0).(CycleMetadata)
}

func (m *MockCycle) SetMetadata(state CycleMetadata) {
	m.Called(state)
}

func (m *MockCycle) TransformToConfig() *CycleConfig {
	args := m.Called()
	return args.Get(0).(*CycleConfig)
}

type MockThrottle struct {
	mock.Mock
}

func (m *MockThrottle) Queue() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockThrottle) Stop() {
	m.Called()
}

func (m *MockThrottle) Interval() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}
