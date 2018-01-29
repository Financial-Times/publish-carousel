package file

import (
	"net/http"
	"strings"
	"testing"

	"errors"

	"github.com/Financial-Times/publish-carousel/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestExternalServiceWithoutBasicAuth(t *testing.T) {
	called := false
	server := cluster.SetupFakeServerNoAuth(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {
		called = true
	})
	defer server.Close()
	watcher := new(cluster.MockWatcher)
	watcher.On("Read", "envsFile").Return("environment1:"+server.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "envsFile", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", http.DefaultClient, "kafka-lagcheck", watcher, "envsFile", "")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.Check()
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestExternalServiceWithBasicAuth(t *testing.T) {
	called := false
	server := cluster.SetupFakeServerBasicAuth(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {
		called = true
	})
	defer server.Close()

	watcher := new(cluster.MockWatcher)
	watcher.On("Read", "envsFile").Return("environment1:"+server.URL, nil)
	watcher.On("Read", "credsFile").Return("environment1:user1:password1", nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "envsFile", mock.AnythingOfType("func(string)"))
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "credsFile", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", http.DefaultClient, "kafka-lagcheck", watcher, "envsFile", "credsFile")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.Check()
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestExternalServiceFails(t *testing.T) {
	called := false
	server := cluster.SetupFakeServerNoAuth(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {
		called = true
	})
	defer server.Close()

	watcher := new(cluster.MockWatcher)
	watcher.On("Read", "envsFile").Return("environment1:"+server.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "envsFile", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", http.DefaultClient, "kafka-lagcheck", watcher, "envsFile", "")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.Check()
	assert.Error(t, err)
	assert.True(t, called)
}

func TestExternalServiceCloses(t *testing.T) {
	server := cluster.SetupFakeServerNoAuth(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {})

	server.Close()

	watcher := new(cluster.MockWatcher)
	watcher.On("Read", "envsFile").Return("environment1:"+server.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "envsFile", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", http.DefaultClient, "kafka-lagcheck", watcher, "envsFile", "")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.Check()
	assert.Error(t, err)
}

func TestSomeExternalServicesFail(t *testing.T) {
	server1 := cluster.SetupFakeServerNoAuth(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := cluster.SetupFakeServerNoAuth(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {})

	defer server1.Close()
	defer server2.Close()

	watcher := new(cluster.MockWatcher)
	watcher.On("Read", "envsFile").Return("environment1:"+server1.URL+",environment2:"+server2.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "envsFile", mock.AnythingOfType("func(string)"))
	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", http.DefaultClient, "kafka-lagcheck", watcher, "envsFile", "")
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
}

func TestSomeExternalServicesSendUnexpectedCodes(t *testing.T) {
	server1 := cluster.SetupFakeServerNoAuth(t, 401, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := cluster.SetupFakeServerNoAuth(t, 504, "/__kafka-lagcheck/__gtg", "", false, func() {})

	defer server1.Close()
	defer server2.Close()

	watcher := new(cluster.MockWatcher)
	watcher.On("Read", "envsFile").Return("environment1:"+server1.URL+",environment2:"+server2.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "envsFile", mock.AnythingOfType("func(string)"))
	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", http.DefaultClient, "kafka-lagcheck", watcher, "envsFile", "")
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
}

func TestMultipleExternalServicesFail(t *testing.T) {
	server1 := cluster.SetupFakeServerNoAuth(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := cluster.SetupFakeServerNoAuth(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})

	defer server1.Close()
	defer server2.Close()

	watcher := new(cluster.MockWatcher)
	watcher.On("Read", "envsFile").Return("environment1:"+server1.URL+",environment2:"+server2.URL, nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "envsFile", mock.AnythingOfType("func(string)"))
	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", http.DefaultClient, "kafka-lagcheck", watcher, "envsFile", "")
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
}

func TestExternalServiceNameAndString(t *testing.T) {
	watcher := new(cluster.MockWatcher)
	watcher.On("Read", "envsFile").Return("environment:localhost", nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "envsFile", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", http.DefaultClient, "kafka-lagcheck", watcher, "envsFile", "")
	assert.NoError(t, err)
	assert.Equal(t, "kafka-lagcheck-delivery", kafkaLagcheck.Name())
	assert.Equal(t, "kafka-lagcheck-delivery - environment: localhost,", kafkaLagcheck.String())
}

func TestExternalServiceConstructorFail(t *testing.T) {
	watcher := new(cluster.MockWatcher)
	watcher.On("Read", "envsFile").Return("", errors.New("read error"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", http.DefaultClient, "kafka-lagcheck", watcher, "envsFile", "")
	assert.NotNil(t, err)
	assert.Nil(t, kafkaLagcheck)
}

func TestExternalServiceCheckInvalidURL(t *testing.T) {
	watcher := new(cluster.MockWatcher)
	watcher.On("Read", "envsFile").Return("environment1:localhost:not-a-number", nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "envsFile", mock.AnythingOfType("func(string)"))

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", http.DefaultClient, "kafka-lagcheck", watcher, "envsFile", "")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.Check()
	assert.NotNil(t, err)
}

func TestExternalServiceClosesRespBody(t *testing.T) {
	c := &cluster.MockClient{}
	watcher := new(cluster.MockWatcher)
	watcher.On("Read", "envsFile").Return("environment1:localhost:not-a-number", nil)
	watcher.On("Watch", mock.AnythingOfType("*context.emptyCtx"), "envsFile", mock.AnythingOfType("func(string)"))

	s, err := NewExternalService("kafka-lagcheck-delivery", c, "kafka-lagcheck", watcher, "envsFile", "")
	assert.NoError(t, err)

	body := &cluster.MockBody{Reader: strings.NewReader(`OK`)}
	resp := &http.Response{Body: body, StatusCode: http.StatusOK}

	c.On("Do", mock.AnythingOfType("*http.Request")).Return(resp, nil)
	body.On("Close").Return(nil)

	assert.NoError(t, s.Check(), "The service should be healthy")
	mock.AssertExpectationsForObjects(t, c, watcher, body)
}
