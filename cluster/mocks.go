package cluster

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/husobee/vestigo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockBody struct {
	mock.Mock
	Reader io.Reader
}

func (b *MockBody) Close() error {
	args := b.Called()
	return args.Error(0)
}

func (b *MockBody) Read(p []byte) (n int, err error) {
	b.Called()
	return b.Reader.Read(p)
}

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

type MockClient struct {
	mock.Mock
}

func (c *MockClient) Do(r *http.Request) (*http.Response, error) {
	args := c.Called(r)
	return args.Get(0).(*http.Response), args.Error(1)
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

type MockWatcher struct {
	mock.Mock
}

func (m *MockWatcher) Watch(ctx context.Context, key string, callback func(val string)) {
	m.Called(ctx, key, callback)
}
func (m *MockWatcher) Read(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}
