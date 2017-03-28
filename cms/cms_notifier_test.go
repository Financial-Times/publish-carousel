package cms

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/publish-carousel/native"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func (m *mockNotifierServer) startMockNotifierServer(t *testing.T) *httptest.Server {
	r := mux.NewRouter()
	r.HandleFunc("/notify", func(w http.ResponseWriter, r *http.Request) {
		tid := r.Header.Get("X-Request-Id")
		hash := r.Header.Get("X-Native-Hash")
		origin := r.Header.Get("X-Origin-System-Id")
		contentType := r.Header.Get("Content-Type")

		body := make(map[string]interface{})
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&body)

		assert.NoError(t, err)

		uuid, ok := body["uuid"]
		assert.NotNil(t, uuid)
		assert.True(t, ok)

		w.WriteHeader(m.Notify(origin, tid, hash, contentType))
	}).Methods("POST")

	r.HandleFunc("/__gtg", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(m.GTG())
	})

	return httptest.NewServer(r)
}

func (m *mockNotifierServer) GTG() int {
	args := m.Called()
	return args.Int(0)
}

func (m *mockNotifierServer) Notify(origin string, tid string, hash string, contentType string) int {
	args := m.Called(origin, tid, hash, contentType)
	return args.Int(0)
}

type mockNotifierServer struct {
	mock.Mock
}

func TestNotify(t *testing.T) {
	mockNotifier := new(mockNotifierServer)
	mockNotifier.On("Notify", "origin", "tid_1234", "12345", "application/json").Return(200)

	server := mockNotifier.startMockNotifierServer(t)

	notifier := NewNotifier(server.URL+"/notify", server.URL+"/__gtg", &http.Client{})

	err := notifier.Notify("origin", "tid_1234", &native.Content{Body: map[string]interface{}{"uuid": "uuid"}, ContentType: "application/json"}, "12345")
	assert.NoError(t, err)
	mockNotifier.AssertExpectations(t)
}

func TestNotifyFails(t *testing.T) {
	mockNotifier := new(mockNotifierServer)
	mockNotifier.On("Notify", "origin", "tid_1234", "12345", "application/json").Return(500)

	server := mockNotifier.startMockNotifierServer(t)

	notifier := NewNotifier(server.URL+"/notify", server.URL+"/__gtg", &http.Client{})

	err := notifier.Notify("origin", "tid_1234", &native.Content{Body: map[string]interface{}{"uuid": "uuid"}, ContentType: "application/json"}, "12345")
	assert.Error(t, err)
	mockNotifier.AssertExpectations(t)
}

func TestNotifierNotRunning(t *testing.T) {
	notifier := NewNotifier("http://localhost/notify", "http://localhost/__gtg", &http.Client{})

	err := notifier.Notify("origin", "tid_1234", &native.Content{}, "12345")
	assert.Error(t, err)
}

func TestJSONFails(t *testing.T) {
	notifier := NewNotifier("http://localhost/notify", "http://localhost/__gtg", &http.Client{})

	body := make(map[string]interface{})
	body["error"] = func() {}
	err := notifier.Notify("origin", "tid_1234", &native.Content{Body: body}, "12345")
	assert.Error(t, err)
}

func TestOKGTG(t *testing.T) {
	mockNotifier := new(mockNotifierServer)
	mockNotifier.On("GTG").Return(200)

	server := mockNotifier.startMockNotifierServer(t)

	notifier := NewNotifier(server.URL+"/notify", server.URL+"/__gtg", &http.Client{})

	err := notifier.GTG()
	assert.NoError(t, err)
	mockNotifier.AssertExpectations(t)
}

func TestFailingGTG(t *testing.T) {
	mockNotifier := new(mockNotifierServer)
	mockNotifier.On("GTG").Return(500)

	server := mockNotifier.startMockNotifierServer(t)

	notifier := NewNotifier(server.URL+"/notify", server.URL+"/__gtg", &http.Client{})

	err := notifier.GTG()
	assert.Error(t, err)
	mockNotifier.AssertExpectations(t)
}

func TestNoServer(t *testing.T) {
	notifier := NewNotifier("http://localhost/notify", "http://localhost/__gtg", &http.Client{})

	err := notifier.GTG()
	assert.Error(t, err)
}
