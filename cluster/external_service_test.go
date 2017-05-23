package cluster

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/husobee/vestigo"
	"github.com/stretchr/testify/assert"
)

func setupFakeServer(t *testing.T, status int, path string, body string, isJSON bool, called func()) *httptest.Server {
	r := vestigo.NewRouter()
	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, path, r.URL.Path)
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

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", "environment:"+server.URL, "")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.Check()
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestExternalServiceFails(t *testing.T) {
	called := false
	server := setupFakeServer(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {
		called = true
	})
	defer server.Close()

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", "environment:"+server.URL, "")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.Check()
	assert.Error(t, err)
	assert.True(t, called)
}

func TestExternalServiceCloses(t *testing.T) {
	server := setupFakeServer(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {})

	server.Close()

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", "environment:"+server.URL, "")
	assert.NoError(t, err)

	name := kafkaLagcheck.ServiceName()
	assert.Equal(t, "kafka-lagcheck", name)

	err = kafkaLagcheck.Check()
	assert.Error(t, err)
}

func TestSomeExternalServicesFail(t *testing.T) {
	server1 := setupFakeServer(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := setupFakeServer(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {})

	defer server1.Close()
	defer server2.Close()

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", "environment:"+server1.URL+",environment2:"+server2.URL, "")
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
	server1 := setupFakeServer(t, 401, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := setupFakeServer(t, 504, "/__kafka-lagcheck/__gtg", "", false, func() {})

	defer server1.Close()
	defer server2.Close()

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", "environment:"+server1.URL+",environment2:"+server2.URL, "")
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
	server1 := setupFakeServer(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := setupFakeServer(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})

	defer server1.Close()
	defer server2.Close()

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", "environment:"+server1.URL+",environment2:"+server2.URL, "")
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
	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", "environment:localhost", "")
	assert.NoError(t, err)
	assert.Equal(t, "kafka-lagcheck-delivery", kafkaLagcheck.Name())
	assert.Equal(t, "kafka-lagcheck-delivery - environment: localhost,", kafkaLagcheck.String())
}
