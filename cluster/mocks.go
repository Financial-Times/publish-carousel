package cluster

import (
	"github.com/stretchr/testify/mock"
	"github.com/husobee/vestigo"
	"github.com/stretchr/testify/assert"
	"testing"
	"net/http/httptest"
	"net/http"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) Check() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockService) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockService) ServiceName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockService) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockService) Description() string {
	args := m.Called()
	return args.String(0)
}

func setupFakeServer(t *testing.T, status int, path string, body string, isJSON bool, usingBasicAuth bool, called func()) *httptest.Server {
	r := vestigo.NewRouter()
	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		username, password, basicAuthHeaderPresent := r.BasicAuth()
		if usingBasicAuth {
			assert.True(t, basicAuthHeaderPresent)
			assert.NotEmpty(t, username)
			assert.NotEmpty(t, password)
		} else {
			assert.False(t, basicAuthHeaderPresent)
		}
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, path, r.URL.Path)
		assert.Equal(t, "UPP Publish Carousel", r.Header.Get("User-Agent"), "user-agent header")

		called()

		if isJSON {
			w.Header().Add("Content-Type", "application/json")
		}

		w.WriteHeader(status)
		w.Write([]byte(body))
	})

	return httptest.NewServer(r)
}

func SetupFakeServerNoAuth(t *testing.T, status int, path string, body string, isJSON bool, called func()) *httptest.Server {
	return setupFakeServer(t, status, path, body, isJSON, false, called)
}

func SetupFakeServerBasicAuth(t *testing.T, status int, path string, body string, isJSON bool, called func()) *httptest.Server {
	return setupFakeServer(t, status, path, body, isJSON, true, called)
}