package cluster

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Financial-Times/publish-carousel/etcd"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupFakeGTG(t *testing.T, status int, path string, called func()) *httptest.Server {
	r := mux.NewRouter()
	r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
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
	watcher.On("Read", "creds-key").Return("environment:user:pass,", nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "creds-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck", watcher, "read-key", "creds-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.Name()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.GTG()
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestExternalServiceFails(t *testing.T) {
	called := false
	server := setupFakeGTG(t, 503, "/__kafka-lagcheck/__gtg", func() {
		called = true
	})
	defer server.Close()

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server.URL, nil)
	watcher.On("Read", "creds-key").Return("environment:user:pass,", nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "creds-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck", watcher, "read-key", "creds-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.Name()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.GTG()
	assert.Error(t, err)
	assert.True(t, called)
}

func TestExternalServiceNoURL(t *testing.T) {
	// if testing.Short() {
	// t.Skip("etcd integration test")
	// }

	readKey, credsKey := etcdKeys("external-service")
	watcher := setupTests(t, readKey, credsKey)

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck", watcher, readKey, credsKey)
	assert.NoError(t, err)

	go func() {
		time.Sleep(500 * time.Millisecond)
		api.Set(context.TODO(), readKey, "", nil)
		api.Set(context.TODO(), credsKey, "environment:user:pass", nil)
	}()

	ctx, cancel := context.WithCancel(context.TODO())
	watcher.Watch(ctx, credsKey, func(val string) {
		assert.Equal(t, "environment:user:pass", val)
		cancel()
	})

	name := kafkaLagcheck.Name()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.GTG()
	assert.NoError(t, err)

	envs := kafkaLagcheck.(*externalService).readService.GetReadEnvironments()
	assert.Len(t, envs, 1)
}

func TestExternalServiceCloses(t *testing.T) {
	server := setupFakeGTG(t, 200, "/__kafka-lagcheck/__gtg", func() {})

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server.URL, nil)
	watcher.On("Read", "creds-key").Return("environment:user:pass,", nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "creds-key", mock.AnythingOfType("func(string)"))

	server.Close()

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck", watcher, "read-key", "creds-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.Name()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.GTG()
	assert.Error(t, err)
}

func TestMultipleExternalServicesFail(t *testing.T) {
	server1 := setupFakeGTG(t, 503, "/__kafka-lagcheck/__gtg", func() {})
	server2 := setupFakeGTG(t, 503, "/__kafka-lagcheck/__gtg", func() {})

	defer server1.Close()
	defer server2.Close()

	watcher := new(etcd.MockWatcher)
	watcher.On("Read", "read-key").Return("environment:"+server1.URL+",environment2:"+server2.URL, nil)
	watcher.On("Read", "creds-key").Return("environment:user:pass,environment2:user:pass", nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "read-key", mock.AnythingOfType("func(string)"))
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "creds-key", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck", watcher, "read-key", "creds-key")
	assert.NoError(t, err)

	name := kafkaLagcheck.Name()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.GTG()
	assert.Error(t, err)

	t.Log(err.Error())
	assert.Contains(t, err.Error(), server1.URL)
	assert.Contains(t, err.Error(), server2.URL)
}
