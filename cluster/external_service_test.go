package cluster

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/publish-carousel/etcd"
	"github.com/husobee/vestigo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupFakeServer(t *testing.T, status int, path string, body string, isJSON bool, called func()) *httptest.Server {
	r := vestigo.NewRouter()
	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
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

func TestExternalService(t *testing.T) {
	called := false
	server := setupFakeServer(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {
		called = true
	})
	defer server.Close()

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.Check()
	assert.NoError(t, err)
	assert.True(t, called)
	watcher.AssertExpectations(t)
}

func TestExternalServiceFails(t *testing.T) {
	called := false
	server := setupFakeServer(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {
		called = true
	})
	defer server.Close()

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.Check()
	assert.Error(t, err)
	assert.True(t, called)
	watcher.AssertExpectations(t)
}

func TestExternalServiceCloses(t *testing.T) {
	server := setupFakeServer(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {})

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	server.Close()

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.Check()
	assert.Error(t, err)
	watcher.AssertExpectations(t)
}

func TestSomeExternalServicesFail(t *testing.T) {
	server1 := setupFakeServer(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := setupFakeServer(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {})

	defer server1.Close()
	defer server2.Close()

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server1.URL+",environment2:"+server2.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	url := kafkaLagcheck.String()
	t.Log(url)

	err = kafkaLagcheck.Check()
	assert.Error(t, err)

	t.Log(err.Error())
	assert.Contains(t, err.Error(), server1.URL)
	assert.NotContains(t, err.Error(), server2.URL)
	watcher.AssertExpectations(t)
}

func TestSomeExternalServicesSendUnexpectedCodes(t *testing.T) {
	server1 := setupFakeServer(t, 401, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := setupFakeServer(t, 504, "/__kafka-lagcheck/__gtg", "", false, func() {})

	defer server1.Close()
	defer server2.Close()

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server1.URL+",environment2:"+server2.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	url := kafkaLagcheck.String()
	t.Log(url)

	err = kafkaLagcheck.Check()
	assert.Error(t, err)

	t.Log(err.Error())
	assert.Contains(t, err.Error(), server1.URL)
	assert.Contains(t, err.Error(), server2.URL)
	watcher.AssertExpectations(t)
}

func TestMultipleExternalServicesFail(t *testing.T) {
	server1 := setupFakeServer(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := setupFakeServer(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})

	defer server1.Close()
	defer server2.Close()

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server1.URL+",environment2:"+server2.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	url := kafkaLagcheck.String()
	t.Log(url)

	err = kafkaLagcheck.Check()
	assert.Error(t, err)

	t.Log(err.Error())
	assert.Contains(t, err.Error(), server1.URL)
	assert.Contains(t, err.Error(), server2.URL)
	watcher.AssertExpectations(t)
}

func TestExternalServiceNameAndString(t *testing.T) {
	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:localhost", nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)
	assert.Equal(t, "kafka-lagcheck-delivery", kafkaLagcheck.Name())
	assert.Equal(t, "kafka-lagcheck-delivery - environment: localhost,", kafkaLagcheck.String())
}
