package resources

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Financial-Times/publish-carousel/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetCycles(t *testing.T) {
	sched := new(scheduler.MockScheduler)
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
	sched := new(scheduler.MockScheduler)
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
	sched := new(scheduler.MockScheduler)
	cycles := make(map[string]scheduler.Cycle)

	sched.On("Cycles").Return(cycles)

	req := httptest.NewRequest("GET", "/cycles/hello", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	sched.AssertExpectations(t)
}

func TestDeleteCycle(t *testing.T) {
	sched := new(scheduler.MockScheduler)
	sched.On("DeleteCycle", "hello").Return(nil)

	req := httptest.NewRequest("DELETE", "/cycles/hello", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	sched.AssertExpectations(t)
}

func TestDeleteCycleFails(t *testing.T) {
	sched := new(scheduler.MockScheduler)
	sched.On("DeleteCycle", "hello").Return(errors.New(`nope didn't delete`))

	req := httptest.NewRequest("DELETE", "/cycles/hello", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	sched.AssertExpectations(t)
}

func TestResumeCycle(t *testing.T) {
	sched := new(scheduler.MockScheduler)
	cycle := new(scheduler.MockCycle)

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
	sched := new(scheduler.MockScheduler)

	cycles := make(map[string]scheduler.Cycle)

	sched.On("Cycles").Return(cycles)

	req := httptest.NewRequest("POST", "/cycles/hello/resume", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	sched.AssertExpectations(t)
}

func TestResetCycle(t *testing.T) {
	sched := new(scheduler.MockScheduler)
	cycle := new(scheduler.MockCycle)

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
	sched := new(scheduler.MockScheduler)

	cycles := make(map[string]scheduler.Cycle)

	sched.On("Cycles").Return(cycles)

	req := httptest.NewRequest("POST", "/cycles/hello/reset", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	sched.AssertExpectations(t)
}

func TestStopCycle(t *testing.T) {
	sched := new(scheduler.MockScheduler)
	cycle := new(scheduler.MockCycle)

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
	sched := new(scheduler.MockScheduler)

	cycles := make(map[string]scheduler.Cycle)

	sched.On("Cycles").Return(cycles)

	req := httptest.NewRequest("POST", "/cycles/hello/stop", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	sched.AssertExpectations(t)
}

func TestGetCycleThrottle(t *testing.T) {
	sched := new(scheduler.MockScheduler)
	cycles := make(map[string]scheduler.Cycle)

	throttle, _ := scheduler.NewThrottle(30*time.Second, 1)
	cycle := scheduler.ThrottledWholeCollectionCycle{Throttle: throttle}
	cycles["123"] = &cycle

	sched.On("Cycles").Return(cycles)

	req := httptest.NewRequest("GET", "/cycles/123/throttle", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"interval":"30s"}`, w.Body.String())
	sched.AssertExpectations(t)
}

func TestCreateCycle(t *testing.T) {
	sched := new(scheduler.MockScheduler)

	name := "test-cycle"

	entity := fmt.Sprintf(`{"name": "%s","type": "ThrottledWholeCollection","origin": "methode-web-pub","collection": "methode","coolDown": "5m0s"}`,
		name)

	expected := scheduler.CycleConfig{
		Name:       name,
		Type:       "ThrottledWholeCollection",
		Origin:     "methode-web-pub",
		Collection: "methode",
		CoolDown:   "5m0s",
		//	Throttle
		//	TimeWindow
		//	MinimumThrottle
		//	MaximumThrottle
	}

	minThrottle := time.Millisecond * 2500
	maxThrottle := time.Millisecond * 500
	cycle := scheduler.NewScalingWindowCycle(name, nil, "test-collection", "methode", time.Second, time.Minute, minThrottle, maxThrottle, nil)
	sched.On("NewCycle", mock.MatchedBy(func(actual scheduler.CycleConfig) bool {
		return reflect.DeepEqual(expected, actual)
	})).Return(cycle, nil)
	sched.On("AddCycle", cycle).Return(nil)

	r := httptest.NewRequest("POST", "/cycles", strings.NewReader(entity))
	w := setupRouter(sched, r)

	assert.Equal(t, http.StatusCreated, w.Code)
	sched.AssertExpectations(t)

	assert.Regexp(t, "/cycles/[0-9a-f]{16}$", w.Header().Get("Location"), "Location header")
}
