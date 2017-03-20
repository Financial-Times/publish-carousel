package resources

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartScheduler(t *testing.T) {
	sched := new(MockScheduler)
	sched.On("Start").Return()

	req := httptest.NewRequest("POST", "/scheduler/start", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusOK, w.Code)
	sched.AssertExpectations(t)
}

func TestShutdownScheduler(t *testing.T) {
	sched := new(MockScheduler)
	sched.On("Shutdown").Return()

	req := httptest.NewRequest("POST", "/scheduler/shutdown", nil)
	w := setupRouter(sched, req)

	assert.Equal(t, http.StatusOK, w.Code)
	sched.AssertExpectations(t)
}
