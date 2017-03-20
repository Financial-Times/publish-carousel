package resources

import (
	"github.com/Financial-Times/publish-carousel/scheduler"
	"github.com/stretchr/testify/mock"
)

type MockScheduler struct {
	mock.Mock
}

func (m *MockScheduler) Cycles() map[string]scheduler.Cycle {
	args := m.Called()
	return args.Get(0).(map[string]scheduler.Cycle)
}
func (m *MockScheduler) Throttles() map[string]scheduler.Throttle {
	args := m.Called()
	return args.Get(0).(map[string]scheduler.Throttle)
}

func (m *MockScheduler) AddThrottle(name string, throttleInterval string) error {
	args := m.Called(name, throttleInterval)
	return args.Error(0)
}

func (m *MockScheduler) DeleteThrottle(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockScheduler) AddCycle(config scheduler.CycleConfig) error {
	args := m.Called(config)
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

func (m *MockScheduler) Start() {
	m.Called()
}

func (m *MockScheduler) Shutdown() {
	m.Called()
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

func (m *MockCycle) Metadata() *scheduler.CycleMetadata {
	args := m.Called()
	return args.Get(0).(*scheduler.CycleMetadata)
}

func (m *MockCycle) RestoreMetadata(state *scheduler.CycleMetadata) {
	m.Called(state)
}

func (m *MockCycle) TransformToConfig() *scheduler.CycleConfig {
	args := m.Called()
	return args.Get(0).(*scheduler.CycleConfig)
}
