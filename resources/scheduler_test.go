package resources

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/publish-carousel/scheduler"
	"github.com/stretchr/testify/assert"
)

func TestStartScheduler(t *testing.T) {
	sched := new(scheduler.MockScheduler)
	sched.On("Start").Return(nil)

	req := httptest.NewRequest("POST", "/scheduler/start", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusOK, w.Code)
	sched.AssertExpectations(t)
}

func TestShutdownScheduler(t *testing.T) {
	sched := new(scheduler.MockScheduler)
	sched.On("Shutdown").Return(nil)

	req := httptest.NewRequest("POST", "/scheduler/shutdown", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusOK, w.Code)
	sched.AssertExpectations(t)
}

func TestUnhappyStartScheduler(t *testing.T) {
	sched := new(scheduler.MockScheduler)
	sched.On("Start").Return(errors.New("life is horrible"))

	req := httptest.NewRequest("POST", "/scheduler/start", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	sched.AssertExpectations(t)
}

func TestUnhappyShutdownScheduler(t *testing.T) {
	sched := new(scheduler.MockScheduler)
	sched.On("Shutdown").Return(errors.New("this southern service has been cancel"))

	req := httptest.NewRequest("POST", "/scheduler/shutdown", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	sched.AssertExpectations(t)
}
