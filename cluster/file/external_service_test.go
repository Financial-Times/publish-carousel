package file

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/Financial-Times/publish-carousel/cluster"
)

func TestExternalServiceWithoutBasicAuth(t *testing.T) {
	called := false
	server := cluster.SetupFakeServerNoAuth(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {
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

func TestExternalServiceWithBasicAuth(t *testing.T) {
	called := false
	server := cluster.SetupFakeServerBasicAuth(t, 200, "/__kafka-lagcheck/__gtg", "", false, func() {
		called = true
	})
	defer server.Close()

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", "environment:"+server.URL, "environment:user:pass")
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

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", "environment:"+server.URL, "")
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

	kafkaLagcheck, err := NewExternalService("kafka-lagcheck-delivery", "kafka-lagcheck", "environment:"+server.URL, "")
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
	server1 := cluster.SetupFakeServerNoAuth(t, 401, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := cluster.SetupFakeServerNoAuth(t, 504, "/__kafka-lagcheck/__gtg", "", false, func() {})

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
	server1 := cluster.SetupFakeServerNoAuth(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})
	server2 := cluster.SetupFakeServerNoAuth(t, 503, "/__kafka-lagcheck/__gtg", "", false, func() {})

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
