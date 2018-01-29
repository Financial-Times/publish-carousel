package cluster

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const pamHealthcheckTemplate = `{"checks":[{"businessImpact":"Publish metrics are not recorded. This will impact the SLA measurement.","checkOutput":"","lastUpdated":"2017-05-05T13:13:23.035614411Z","name":"MessageQueueProxyReachable","ok":true,"panicGuide":"https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/publish-availability-monitor","severity":1,"technicalSummary":"Message queue proxy is not reachable/healthy"},{"businessImpact":"At least two of the last 10 publishes failed. This will reflect in the SLA measurement.","checkOutput":"","lastUpdated":"2017-05-05T13:13:23.035609161Z","name":"ReflectPublishFailures","ok":true,"panicGuide":"https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/publish-availability-monitor","severity":1,"technicalSummary":"Publishes did not meet the SLA measurments"},{"businessImpact":"Publish metrics might not be correct. False positive failures might be recorded. This will impact the SLA measurement.","checkOutput":"","lastUpdated":"2017-05-05T13:13:23.03562623Z","name":"validationServicesReachable","ok":true,"panicGuide":"https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/publish-availability-monitor","severity":1,"technicalSummary":"Validation services are not reachable/healthy"},{"businessImpact":"Publish metrics are not recorded. This will impact the SLA measurement.","checkOutput":"","lastUpdated":"2017-05-05T13:13:23.035881681Z","name":"IsConsumingFromNotificationsPushFeeds","ok":true,"panicGuide":"https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/publish-availability-monitor","severity":1,"technicalSummary":"The connections to the configured notifications-push feeds are operating correctly."},{"businessImpact":"Publish metrics might not be correct. False positive failures might be recorded. This will impact the SLA measurement.","checkOutput":"","lastUpdated":"2017-05-05T13:13:23.035594744Z","name":"loadtest-delivery readEndpointsReachable","ok":true,"panicGuide":"https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/publish-availability-monitor","severity":1,"technicalSummary":"Read services are not reachable/healthy"}],"description":"Checks if all the dependent services are reachable and healthy.","name":"Dependent services healthcheck","schemaVersion":1,"ok":%v}`

func TestHappyNewService(t *testing.T) {
	s, err := NewService("pam", "http://someting.com", false)

	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, "pam", s.ServiceName(), "The name should match the one gave in the constructor function")
}

func TestUnhappyNewService(t *testing.T) {
	_, err := NewService("pam", "a not valid url", false)
	assert.EqualError(t, err, "parse a not valid url/__gtg: invalid URI for request", "It should return an error for invalid URI")
}

func TestGTGClosesConnectionsIfHealthy(t *testing.T) {
	c := &mockClient{}
	gtg, _ := url.Parse("/__gtg")
	health, _ := url.Parse("/__health")
	s := clusterService{client: c, serviceName: "pam", gtgURL: gtg, healthURL: health, checkHealthchecks: false}

	body := &MockBody{Reader: strings.NewReader(`OK`)}
	resp := &http.Response{Body: body, StatusCode: http.StatusOK}

	c.On("Do", mock.AnythingOfType("*http.Request")).Return(resp, nil)
	body.On("Close").Return(nil)

	assert.NoError(t, s.Check(), "The service should be good to go")
	mock.AssertExpectationsForObjects(t, c, body)
}

func TestGTGClosesConnectionsIfUnhealthy(t *testing.T) {
	c := &mockClient{}
	gtg, _ := url.Parse("/__gtg")
	health, _ := url.Parse("/__health")
	s := clusterService{client: c, serviceName: "pam", gtgURL: gtg, healthURL: health, checkHealthchecks: false}

	body := &MockBody{Reader: strings.NewReader(`OK`)}
	resp := &http.Response{Body: body, StatusCode: http.StatusForbidden}

	c.On("Do", mock.AnythingOfType("*http.Request")).Return(resp, nil)
	body.On("Close").Return(nil)

	assert.Error(t, s.Check(), "The service should be good to go")
	mock.AssertExpectationsForObjects(t, c, body)
}

func TestHappyGTG(t *testing.T) {
	called := false
	mockService := SetupFakeServerNoAuth(t, http.StatusOK, "/__gtg", "", false, func() {
		called = true
	})
	defer mockService.Close()

	s, err := NewService("pam", mockService.URL, false)
	assert.NoError(t, err, "It should not return an error")
	assert.NoError(t, s.Check(), "The service should be good to go")
	assert.True(t, called)
}

func TestUnhappyGTG(t *testing.T) {
	called := false
	mockService := setupFakeServer(t, http.StatusServiceUnavailable, "/__gtg", "", false, false, func() {
		called = true
	})
	defer mockService.Close()

	s, err := NewService("pam", mockService.URL, false)
	assert.NoError(t, err, "It should not return an error")
	assert.EqualError(t, s.Check(), "GTG for pam returned a non-200 code: 503", "The service should not be good to go")
	assert.True(t, called)
}

func TestHappyHealth(t *testing.T) {
	called := false
	mockService := setupFakeServer(t, http.StatusOK, "/__health", fmt.Sprintf(pamHealthcheckTemplate, "true"), true, false, func() {
		called = true
	})
	defer mockService.Close()

	s, err := NewService("pam", mockService.URL, true)
	assert.NoError(t, err)

	assert.NoError(t, s.Check())
	assert.True(t, called)
}

func TestUnhappyHealth(t *testing.T) {
	called := false
	mockService := setupFakeServer(t, http.StatusOK, "/__health", fmt.Sprintf(pamHealthcheckTemplate, "false"), true, false, func() {
		called = true
	})
	defer mockService.Close()

	s, err := NewService("pam", mockService.URL, true)
	assert.NoError(t, err)
	assert.EqualError(t, s.Check(), fmt.Sprintf(`Healthcheck failed @ "%v/__health"`, mockService.URL))
	assert.True(t, called)
}

func TestEmptyHealthResponse(t *testing.T) {
	called := false
	mockService := setupFakeServer(t, http.StatusOK, "/__health", "", true, false, func() {
		called = true
	})
	defer mockService.Close()

	s, err := NewService("pam", mockService.URL, true)
	assert.NoError(t, err)

	assert.EqualError(t, s.Check(), "EOF")
	assert.True(t, called)
}

func TestNonOKHealth(t *testing.T) {
	called := false
	mockService := setupFakeServer(t, http.StatusServiceUnavailable, "/__health", fmt.Sprintf(pamHealthcheckTemplate, "false"), true, false, func() {
		called = true
	})
	defer mockService.Close()

	s, err := NewService("pam", mockService.URL, true)
	assert.NoError(t, err)
	assert.EqualError(t, s.Check(), fmt.Sprintf(`Non-200 code while checking "%v/__health"`, mockService.URL))
	assert.True(t, called)
}

func TestHealthConnection(t *testing.T) {
	s, err := NewService("pam", "http://a-url-that-does-not-exixts.com/something", true)
	assert.NoError(t, err)
	assert.Error(t, s.Check())
}

func TestGTGConnectionError(t *testing.T) {
	s, err := NewService("pam", "http://a-url-that-does-not-exixts.com/something", false)

	assert.NoError(t, err, "It should not return an error")
	assert.Error(t, s.Check(), "The service should not be good to go")
}

func TestServiceNameAndString(t *testing.T) {
	s, err := NewService("pam", "http://a-url-that-does-not-exixts.com/something", false)
	assert.NoError(t, err, "It should not return an error")

	assert.Equal(t, "pam", s.Name())
	assert.Equal(t, "http://a-url-that-does-not-exixts.com/something/__gtg", s.String())
}

type mockClient struct {
	mock.Mock
}

func (c *mockClient) Do(r *http.Request) (*http.Response, error) {
	args := c.Called(r)
	return args.Get(0).(*http.Response), args.Error(1)
}
