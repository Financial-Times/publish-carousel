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

func setupFakeGTG(t *testing.T, status int, path string, called func()) *httptest.Server {
	r := vestigo.NewRouter()
	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, path, r.URL.Path)
		called()

		w.WriteHeader(status)
	})

	return httptest.NewServer(r)
}

func TestExternalService(t *testing.T) {
	called := false
	server := setupFakeGTG(t, 200, "/__kafka-lagcheck/__gtg", func() {
		called = true
	})
	defer server.Close()

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.Name()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.GTG()
	assert.NoError(t, err)
	assert.True(t, called)
	watcher.AssertExpectations(t)
}

func TestExternalServiceFails(t *testing.T) {
	called := false
	server := setupFakeGTG(t, 503, "/__kafka-lagcheck/__gtg", func() {
		called = true
	})
	defer server.Close()

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.Name()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.GTG()
	assert.Error(t, err)
	assert.True(t, called)
	watcher.AssertExpectations(t)
}

func TestExternalServiceCloses(t *testing.T) {
	server := setupFakeGTG(t, 200, "/__kafka-lagcheck/__gtg", func() {})

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	server.Close()

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.Name()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.GTG()
	assert.Error(t, err)
	watcher.AssertExpectations(t)
}

func TestSomeExternalServicesFail(t *testing.T) {
	server1 := setupFakeGTG(t, 503, "/__kafka-lagcheck/__gtg", func() {})
	server2 := setupFakeGTG(t, 200, "/__kafka-lagcheck/__gtg", func() {})

	defer server1.Close()
	defer server2.Close()

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server1.URL+",environment2:"+server2.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.Name()
	assert.Equal(t, "kafka-lagcheck", name)

	url := kafkaLagcheck.Description()
	t.Log(url)

	err = kafkaLagcheck.GTG()
	assert.Error(t, err)

	t.Log(err.Error())
	assert.Contains(t, err.Error(), server1.URL)
	assert.NotContains(t, err.Error(), server2.URL)
	watcher.AssertExpectations(t)
}

func TestSomeExternalServicesSendUnexpectedCodes(t *testing.T) {
	server1 := setupFakeGTG(t, 401, "/__kafka-lagcheck/__gtg", func() {})
	server2 := setupFakeGTG(t, 504, "/__kafka-lagcheck/__gtg", func() {})

	defer server1.Close()
	defer server2.Close()

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server1.URL+",environment2:"+server2.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.Name()
	assert.Equal(t, "kafka-lagcheck", name)

	url := kafkaLagcheck.Description()
	t.Log(url)

	err = kafkaLagcheck.GTG()
	assert.Error(t, err)

	t.Log(err.Error())
	assert.Contains(t, err.Error(), server1.URL)
	assert.Contains(t, err.Error(), server2.URL)
	watcher.AssertExpectations(t)
}

func TestMultipleExternalServicesFail(t *testing.T) {
	server1 := setupFakeGTG(t, 503, "/__kafka-lagcheck/__gtg", func() {})
	server2 := setupFakeGTG(t, 503, "/__kafka-lagcheck/__gtg", func() {})

	defer server1.Close()
	defer server2.Close()

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server1.URL+",environment2:"+server2.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck", watcher, "read-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.Name()
	assert.Equal(t, "kafka-lagcheck", name)

	url := kafkaLagcheck.Description()
	t.Log(url)

	err = kafkaLagcheck.GTG()
	assert.Error(t, err)

	t.Log(err.Error())
	assert.Contains(t, err.Error(), server1.URL)
	assert.Contains(t, err.Error(), server2.URL)
	watcher.AssertExpectations(t)
}
