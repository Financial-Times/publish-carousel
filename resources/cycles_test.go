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

	"github.com/Financial-Times/publish-carousel/blacklist"
	"github.com/Financial-Times/publish-carousel/native"
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

	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(nil, nil, blacklist.NoOpBlacklist)

	cycle := scheduler.NewScalingWindowCycle(name, uuidCollectionBuilder, "test-collection", "methode", time.Second, time.Minute, minThrottle, maxThrottle, nil)
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
	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(nil, nil, blacklist.NoOpBlacklist)

	cycle := scheduler.NewThrottledWholeCollectionCycle("test-cycle", uuidCollectionBuilder, "test-collection", "test-origin", time.Minute, throttle, nil)
	cycles["123"] = cycle

	sched.On("Cycles").Return(cycles)

	req := httptest.NewRequest("GET", "/cycles/123/throttle", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"interval":"30s"}`, w.Body.String())
	sched.AssertExpectations(t)
}

func TestSetCycleThrottle(t *testing.T) {
	name := "test-cycle"
	origin := "methode-web-pub"
	collection := "test-collection"
	oldThrottle, _ := scheduler.NewThrottle(30*time.Second, 1)

	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(nil, nil, blacklist.NoOpBlacklist)

	oldCycle := scheduler.NewThrottledWholeCollectionCycle(name, uuidCollectionBuilder, collection, origin, time.Minute, oldThrottle, nil)
	cycleID := oldCycle.ID()

	metadata := scheduler.CycleMetadata{
		CurrentPublishUUID: "00000000-0000-0000-0000-000000000000",
		Errors:             1,
		Progress:           0.5,
		State:              []string{"running", "healthy"},
		Completed:          2,
		Total:              3,
		Iteration:          4,
		Start:              nil,
		End:                nil,
	}
	oldCycle.SetMetadata(metadata)

	throttleEntity := `{"interval": "10s"}`
	newThrottle, _ := scheduler.NewThrottle(10*time.Second, 1)

	sched := new(scheduler.MockScheduler)
	cycles := make(map[string]scheduler.Cycle)

	cycles[cycleID] = oldCycle

	sched.On("Cycles").Return(cycles)
	sched.On("DeleteCycle", cycleID).Return(nil)

	expected := scheduler.CycleConfig{
		Name:       name,
		Type:       "ThrottledWholeCollection",
		Origin:     origin,
		Collection: collection,
		CoolDown:   "1m0s",
		Throttle:   "10s",
		//	TimeWindow
		//	MinimumThrottle
		//	MaximumThrottle
	}

	newCycle := scheduler.NewThrottledWholeCollectionCycle(name, uuidCollectionBuilder, collection, origin, time.Minute, newThrottle, nil)
	sched.On("NewCycle", mock.MatchedBy(func(actual scheduler.CycleConfig) bool {
		return reflect.DeepEqual(expected, actual)
	})).Return(newCycle, nil)
	sched.On("AddCycle", newCycle).Return(nil)

	req := httptest.NewRequest("PUT", fmt.Sprintf("/cycles/%s/throttle", cycleID), strings.NewReader(throttleEntity))
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	sched.AssertExpectations(t)
	assert.Regexp(t, fmt.Sprintf("/cycles/%s$", cycleID), w.Header().Get("Location"), "Location header")
	assert.Equal(t, newCycle.Metadata(), metadata)
}

func TestSetCycleThrottleRedirectUsesOriginalRequestURL(t *testing.T) {
	name := "test-cycle"
	origin := "methode-web-pub"
	collection := "test-collection"
	oldThrottle, _ := scheduler.NewThrottle(30*time.Second, 1)
	uuidCollectionBuilder := native.NewNativeUUIDCollectionBuilder(nil, nil, blacklist.NoOpBlacklist)

	oldCycle := scheduler.NewThrottledWholeCollectionCycle(name, uuidCollectionBuilder, collection, origin, time.Minute, oldThrottle, nil)
	cycleID := oldCycle.ID()

	metadata := scheduler.CycleMetadata{
		CurrentPublishUUID: "00000000-0000-0000-0000-000000000000",
		Errors:             1,
		Progress:           0.5,
		State:              []string{"running", "healthy"},
		Completed:          2,
		Total:              3,
		Iteration:          4,
		Start:              nil,
		End:                nil,
	}
	oldCycle.SetMetadata(metadata)

	throttleEntity := `{"interval": "10s"}`
	newThrottle, _ := scheduler.NewThrottle(10*time.Second, 1)

	sched := new(scheduler.MockScheduler)
	cycles := make(map[string]scheduler.Cycle)

	cycles[cycleID] = oldCycle

	sched.On("Cycles").Return(cycles)
	sched.On("DeleteCycle", cycleID).Return(nil)

	expected := scheduler.CycleConfig{
		Name:       name,
		Type:       "ThrottledWholeCollection",
		Origin:     origin,
		Collection: collection,
		CoolDown:   "1m0s",
		Throttle:   "10s",
		//	TimeWindow
		//	MinimumThrottle
		//	MaximumThrottle
	}

	newCycle := scheduler.NewThrottledWholeCollectionCycle(name, uuidCollectionBuilder, collection, origin, time.Minute, newThrottle, nil)
	sched.On("NewCycle", mock.MatchedBy(func(actual scheduler.CycleConfig) bool {
		return reflect.DeepEqual(expected, actual)
	})).Return(newCycle, nil)
	sched.On("AddCycle", newCycle).Return(nil)

	setThrottlePath := fmt.Sprintf("/cycles/%s/throttle", cycleID)
	req := httptest.NewRequest("PUT", setThrottlePath, strings.NewReader(throttleEntity))
	req.Header.Set("X-Original-Request-URL", "https://www.example.com/__test/"+setThrottlePath)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusSeeOther, w.Code)
	sched.AssertExpectations(t)
	assert.Equal(t, "https://www.example.com/__test/"+fmt.Sprintf("/cycles/%s", cycleID), w.Header().Get("Location"), "Location header")
	assert.Equal(t, newCycle.Metadata(), metadata)
}
