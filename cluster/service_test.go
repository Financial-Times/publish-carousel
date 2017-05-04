package cluster

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/stretchr/testify/assert"
)

func newMockHTTPService(t *testing.T, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(status)
		assert.Contains(t, req.URL.Path, httphandlers.GTGPath, "Request URL should contain GTG path")
	}))
}

func TestHappyNewService(t *testing.T) {
	s, err := NewService("pam", "http://someting.com")
	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, "pam", s.ServiceName(), "The name should match the one gave in the constructor function")
}

func TestUnhappyNewService(t *testing.T) {
	_, err := NewService("pam", "a not valid url")
	assert.EqualError(t, err, "parse a not valid url/__gtg: invalid URI for request", "It should return an error for invalid URI")
}

func TestHappyGTG(t *testing.T) {
	mockService := newMockHTTPService(t, http.StatusOK)
	defer mockService.Close()

	s, err := NewService("pam", mockService.URL)
	assert.NoError(t, err, "It should not return an error")
	assert.NoError(t, s.GTG(), "The service should be good to go")
}

func TestUnhappyGTG(t *testing.T) {
	mockService := newMockHTTPService(t, http.StatusServiceUnavailable)
	defer mockService.Close()

	s, err := NewService("pam", mockService.URL)
	assert.NoError(t, err, "It should not return an error")
	assert.EqualError(t, s.GTG(), "GTG for pam returned a non-200 code: 503", "The service should not be good to go")
}

func TestGTGConnectionError(t *testing.T) {
	s, err := NewService("pam", "http://a-url-that-does-not-exixts.com/something")
	assert.NoError(t, err, "It should not return an error")
	assert.Error(t, s.GTG(), "The service should not be good to go")
}

func TestServiceNameAndString(t *testing.T) {
	s, err := NewService("pam", "http://a-url-that-does-not-exixts.com/something")
	assert.NoError(t, err, "It should not return an error")

	assert.Equal(t, "pam", s.Name())
	assert.Equal(t, "http://a-url-that-does-not-exixts.com/something/__gtg", s.String())
}
