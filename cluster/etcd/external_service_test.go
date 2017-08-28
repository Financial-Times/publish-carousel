package etcd

import (
	"testing"

	"github.com/Financial-Times/publish-carousel/etcd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/Financial-Times/publish-carousel/cluster"
)

func TestExternalService(t *testing.T) {
	called := false
	server := cluster.SetupFakeServerNoAuth(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {
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
	server := cluster.SetupFakeServerNoAuth(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {
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
	server := cluster.SetupFakeServerNoAuth(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {})

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
	server1 := cluster.SetupFakeServerNoAuth(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := cluster.SetupFakeServerNoAuth(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {})

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
	server1 := cluster.SetupFakeServerNoAuth(t, 401, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := cluster.SetupFakeServerNoAuth(t, 504, "/__kafka-lagcheck/__gtg", "", false, func() {})

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
	server1 := cluster.SetupFakeServerNoAuth(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := cluster.SetupFakeServerNoAuth(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})

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
