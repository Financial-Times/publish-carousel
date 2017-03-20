package resources

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/publish-carousel/scheduler"
	"github.com/stretchr/testify/assert"
)

func TestGetCycles(t *testing.T) {
	sched := new(MockScheduler)
	cycles := make(map[string]scheduler.Cycle)

	cycle := scheduler.ScalingWindowCycle{MaximumThrottle: `maxThrottle`}
	cycles["hello"] = &cycle

	sched.On("Cycles").Return(cycles)

	r := httptest.NewRequest("GET", "/cycles", nil)
	w := setupRouter(sched, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `[{"maximumThrottle":"maxThrottle"}]`, w.Body.String())
	sched.AssertExpectations(t)
}

func TestGetCyclesForID(t *testing.T) {
	sched := new(MockScheduler)
	cycles := make(map[string]scheduler.Cycle)

	cycle := scheduler.ScalingWindowCycle{MaximumThrottle: `maxThrottle`}
	cycles["hello"] = &cycle

	sched.On("Cycles").Return(cycles)

	req := httptest.NewRequest("GET", "/cycles/hello", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"maximumThrottle":"maxThrottle"}`, w.Body.String())
	sched.AssertExpectations(t)
}

func TestGetCyclesForIDCycleNotFound(t *testing.T) {
	sched := new(MockScheduler)
	cycles := make(map[string]scheduler.Cycle)

	sched.On("Cycles").Return(cycles)

	req := httptest.NewRequest("GET", "/cycles/hello", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	sched.AssertExpectations(t)
}

func TestDeleteCycle(t *testing.T) {
	sched := new(MockScheduler)
	sched.On("DeleteCycle", "hello").Return(nil)

	req := httptest.NewRequest("DELETE", "/cycles/hello", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	sched.AssertExpectations(t)
}

func TestDeleteCycleFails(t *testing.T) {
	sched := new(MockScheduler)
	sched.On("DeleteCycle", "hello").Return(errors.New(`nope didn't delete`))

	req := httptest.NewRequest("DELETE", "/cycles/hello", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	sched.AssertExpectations(t)
}

func TestResumeCycle(t *testing.T) {
	sched := new(MockScheduler)
	cycle := new(MockCycle)

	cycles := make(map[string]scheduler.Cycle)
	cycles["hello"] = cycle

	sched.On("Cycles").Return(cycles)
	cycle.On("Start").Return()

	req := httptest.NewRequest("POST", "/cycles/hello/resume", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusOK, w.Code)
	sched.AssertExpectations(t)
}

func TestResumeCycleNotFound(t *testing.T) {
	sched := new(MockScheduler)

	cycles := make(map[string]scheduler.Cycle)

	sched.On("Cycles").Return(cycles)

	req := httptest.NewRequest("POST", "/cycles/hello/resume", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	sched.AssertExpectations(t)
}

func TestResetCycle(t *testing.T) {
	sched := new(MockScheduler)
	cycle := new(MockCycle)

	cycles := make(map[string]scheduler.Cycle)
	cycles["hello"] = cycle

	sched.On("Cycles").Return(cycles)
	cycle.On("Reset").Return()

	req := httptest.NewRequest("POST", "/cycles/hello/reset", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusOK, w.Code)
	sched.AssertExpectations(t)
}

func TestResetCycleNotFound(t *testing.T) {
	sched := new(MockScheduler)

	cycles := make(map[string]scheduler.Cycle)

	sched.On("Cycles").Return(cycles)

	req := httptest.NewRequest("POST", "/cycles/hello/reset", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	sched.AssertExpectations(t)
}

func TestStopCycle(t *testing.T) {
	sched := new(MockScheduler)
	cycle := new(MockCycle)

	cycles := make(map[string]scheduler.Cycle)
	cycles["hello"] = cycle

	sched.On("Cycles").Return(cycles)
	cycle.On("Stop").Return()

	req := httptest.NewRequest("POST", "/cycles/hello/stop", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusOK, w.Code)
	sched.AssertExpectations(t)
}

func TestStopCycleNotFound(t *testing.T) {
	sched := new(MockScheduler)

	cycles := make(map[string]scheduler.Cycle)

	sched.On("Cycles").Return(cycles)

	req := httptest.NewRequest("POST", "/cycles/hello/stop", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	sched.AssertExpectations(t)
}
