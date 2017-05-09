package resources

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	fthealth "github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/publish-carousel/cluster"
	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/s3"
	"github.com/Financial-Times/publish-carousel/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupHappyMocks() map[string]interface{} {
	sched := new(scheduler.MockScheduler)

	c1 := new(scheduler.MockCycle)
	c1.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"running"}})
	c2 := new(scheduler.MockCycle)
	c2.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"running"}})

	mockCycles := map[string]scheduler.Cycle{
		"c1": c1,
		"c2": c2,
	}

	sched.On("Cycles").Return(mockCycles)
	sched.On("IsRunning").Return(true)
	sched.On("IsEnabled").Return(true)
	sched.On("IsAutomaticallyDisabled").Return(false)
	sched.On("WasAutomaticallyDisabled").Return(false)

	upService1 := new(cluster.MockService)
	upService1.On("GTG").Return(nil)
	upService2 := new(cluster.MockService)
	upService2.On("GTG").Return(nil)

	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	db.On("Open").Return(mockTx, nil)
	mockTx.On("Ping").Return(nil)
	mockTx.On("Close").Return()

	s3RW := new(s3.MockReadWriter)
	s3RW.On("Ping").Return(nil)

	cmsNotifier := new(cms.MockNotifier)
	cmsNotifier.On("GTG").Return(nil)

	mocks := map[string]interface{}{
		"scheduler":   sched,
		"cycle1":      c1,
		"cycle2":      c2,
		"service1":    upService1,
		"service2":    upService2,
		"db":          db,
		"tx":          mockTx,
		"s3RW":        s3RW,
		"cmsNotifier": cmsNotifier,
	}
	return mocks
}

func setupTestHealthcheckEndpoint(configError error) (func(w http.ResponseWriter, r *http.Request), map[string]interface{}) {
	mocks := setupHappyMocks()
	return Health(
		mocks["db"].(native.DB),
		mocks["s3RW"].(s3.ReadWriter),
		mocks["cmsNotifier"].(cms.Notifier),
		mocks["scheduler"].(scheduler.Scheduler),
		configError,
		mocks["service1"].(cluster.Service),
		mocks["service2"].(cluster.Service),
	), mocks
}

func setupTestGTGEndpoint(configError error) (func(w http.ResponseWriter, r *http.Request), map[string]interface{}) {
	mocks := setupHappyMocks()
	return GTG(
		mocks["db"].(native.DB),
		mocks["s3RW"].(s3.ReadWriter),
		mocks["cmsNotifier"].(cms.Notifier),
		mocks["scheduler"].(scheduler.Scheduler),
		configError,
		mocks["service1"].(cluster.Service),
		mocks["service2"].(cluster.Service),
	), mocks
}

func parseHealthcheck(healthcheckJSON string) ([]fthealth.CheckResult, error) {
	result := &struct {
		Checks []fthealth.CheckResult `json:"checks"`
	}{}

	err := json.Unmarshal([]byte(healthcheckJSON), result)
	return result.Checks, err
}

<<<<<<< HEAD
	sched.On("Cycles").Return(mockCycles)
	sched.On("IsRunning").Return(true)
	sched.On("IsEnabled").Return(true)
	sched.On("IsAutomaticallyDisabled").Return(false)
	sched.On("WasAutomaticallyDisabled").Return(false)
=======
func TestHappyHealthcheck(t *testing.T) {
	endpoint, mocks := setupTestHealthcheckEndpoint(nil)
	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()
>>>>>>> master

	endpoint(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Healthcheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

	for _, check := range checks {
		assert.True(t, check.Ok)
		assert.NotEmpty(t, check.Output)
	}

	for _, m := range mocks {
		mock.AssertExpectationsForObjects(t, m)
	}
}

func TestMongoDBFailsToOpenHealthcheck(t *testing.T) {
	endpoint, mocks := setupTestHealthcheckEndpoint(nil)
	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	db := mocks["db"].(*native.MockDB)
	db.ExpectedCalls = make([]*mock.Call, 0)
	db.On("Open").Return(mocks["tx"], errors.New("oops"))
	delete(mocks, "tx")

	endpoint(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Healthcheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

	for _, check := range checks {
		if check.Name == "CheckConnectivityToNativeDatabase" {
			assert.False(t, check.Ok)
		} else {
			assert.True(t, check.Ok)
		}
	}

<<<<<<< HEAD
	sched.On("Cycles").Return(mockCycles)
	sched.On("IsRunning").Return(true)
	sched.On("IsEnabled").Return(true)
	sched.On("IsAutomaticallyDisabled").Return(false)
	sched.On("WasAutomaticallyDisabled").Return(false)

	upService1 := new(cluster.MockService)
	upService1.On("GTG").Return(nil)
	upService2 := new(cluster.MockService)
	upService2.On("GTG").Return(nil)

	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	db.On("Open").Return(mockTx, nil)
	mockTx.On("Ping").Return(nil)
	mockTx.On("Close").Return()

	s3RW := new(s3.MockReadWriter)
	s3RW.On("Ping").Return(errors.New("amazon data center got fire"))

	cmsNotifier := new(cms.MockNotifier)
	cmsNotifier.On("GTG").Return(nil)
=======
	for _, m := range mocks {
		mock.AssertExpectationsForObjects(t, m)
	}
}
>>>>>>> master

func TestMongoDBFailsToPingHealthcheck(t *testing.T) {
	endpoint, mocks := setupTestHealthcheckEndpoint(nil)
	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	tx := mocks["tx"].(*native.MockTX)
	tx.ExpectedCalls = make([]*mock.Call, 0)

	tx.On("Ping").Return(errors.New("no ping 4 u"))
	tx.On("Close")

	endpoint(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Healthcheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

	for _, check := range checks {
		if check.Name == "CheckConnectivityToNativeDatabase" {
			assert.False(t, check.Ok)
		} else {
			assert.True(t, check.Ok)
		}
	}

<<<<<<< HEAD
	sched.On("Cycles").Return(mockCycles)
	sched.On("IsRunning").Return(true)
	sched.On("IsEnabled").Return(true)
	sched.On("IsAutomaticallyDisabled").Return(false)
	sched.On("WasAutomaticallyDisabled").Return(false)
=======
	for _, m := range mocks {
		mock.AssertExpectationsForObjects(t, m)
	}
}
>>>>>>> master

func TestUnhappyS3Healthcheck(t *testing.T) {
	endpoint, mocks := setupTestHealthcheckEndpoint(nil)
	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	s3RW := mocks["s3RW"].(*s3.MockReadWriter)
	s3RW.ExpectedCalls = make([]*mock.Call, 0)

	s3RW.On("Ping").Return(errors.New("amazon data center got fire"))

	endpoint(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Healthcheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

<<<<<<< HEAD
	Health(db, s3RW, cmsNotifier, sched, nil, upService1, upService2)(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck should return 200")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToNativeDatabase","ok":true`, "Database should be healthy")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToS3","ok":true`, "S3 should be healthy")
	assert.Contains(t, string(body), `"name":"CheckCMSNotifierHealth","ok":false`, "CMS notifier should not be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCycles","ok":true`, "Cycles should be healthy")
	assert.Contains(t, string(body), `"name":"InvalidCycleConfiguration","ok":true`, "Cycles configuration should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCluster","ok":true`, "Cluster should be healthy")
	assert.Contains(t, string(body), `"name":"ActivePublishingCluster","ok":true`, "Cluster should not be in failover state")

	sched.AssertExpectations(t)
	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	s3RW.AssertExpectations(t)
	cmsNotifier.AssertExpectations(t)
	upService1.AssertExpectations(t)
	upService2.AssertExpectations(t)
}
=======
	for _, check := range checks {
		if check.Name == "CheckConnectivityToS3" {
			assert.False(t, check.Ok)
		} else {
			assert.True(t, check.Ok)
		}
	}
>>>>>>> master

	for _, m := range mocks {
		mock.AssertExpectationsForObjects(t, m)
	}
}

func TestUnhappyCMSNotifierHealthcheck(t *testing.T) {
	endpoint, mocks := setupTestHealthcheckEndpoint(nil)
	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	cmsNotifier := mocks["cmsNotifier"].(*cms.MockNotifier)
	cmsNotifier.ExpectedCalls = make([]*mock.Call, 0)

<<<<<<< HEAD
	sched.On("Cycles").Return(mockCycles)
	sched.On("IsRunning").Return(true)
	sched.On("IsEnabled").Return(true)
	sched.On("IsAutomaticallyDisabled").Return(false)
	sched.On("WasAutomaticallyDisabled").Return(false)
=======
	cmsNotifier.On("GTG").Return(errors.New("not available"))
>>>>>>> master

	endpoint(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Healthcheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

	for _, check := range checks {
		if check.Name == "CheckCMSNotifierHealth" {
			assert.False(t, check.Ok)
		} else {
			assert.True(t, check.Ok)
		}
	}

	for _, m := range mocks {
		mock.AssertExpectationsForObjects(t, m)
	}
}

func TestUnhappyCyclesHealthcheck(t *testing.T) {
	endpoint, mocks := setupTestHealthcheckEndpoint(nil)
	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

<<<<<<< HEAD
	Health(db, s3RW, cmsNotifier, sched, nil, upService1, upService2)(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck should return 200")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToNativeDatabase","ok":true`, "Database should be healthy")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToS3","ok":true`, "S3 should be healthy")
	assert.Contains(t, string(body), `"name":"CheckCMSNotifierHealth","ok":true`, "CMS notifier should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCycles","ok":false`, "Cycles should not be healthy")
	assert.Contains(t, string(body), `"name":"InvalidCycleConfiguration","ok":true`, "Cycles configuration should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCluster","ok":true`, "Cluster should be healthy")
	assert.Contains(t, string(body), `"name":"ActivePublishingCluster","ok":true`, "Cluster should not be in failover state")

	sched.AssertExpectations(t)
	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	s3RW.AssertExpectations(t)
	cmsNotifier.AssertExpectations(t)
	upService1.AssertExpectations(t)
	upService2.AssertExpectations(t)
}
=======
	c1 := mocks["cycle1"].(*scheduler.MockCycle)
	c1.ExpectedCalls = make([]*mock.Call, 0)
>>>>>>> master

	c1.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"unhealthy"}})
	c1.On("ID").Return("c1")

	endpoint(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Healthcheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

<<<<<<< HEAD
	sched.On("Cycles").Return(mockCycles)
	sched.On("IsRunning").Return(true)
	sched.On("IsEnabled").Return(true)
	sched.On("IsAutomaticallyDisabled").Return(false)
	sched.On("WasAutomaticallyDisabled").Return(false)
=======
	for _, check := range checks {
		if check.Name == "UnhealthyCycles" {
			assert.False(t, check.Ok)
		} else {
			assert.True(t, check.Ok)
		}
	}
>>>>>>> master

	for _, m := range mocks {
		mock.AssertExpectationsForObjects(t, m)
	}
}

func TestUnhappyCycleConfigHealthcheck(t *testing.T) {
	endpoint, mocks := setupTestHealthcheckEndpoint(errors.New("something wrong happened"))
	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	endpoint(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Healthcheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

	for _, check := range checks {
		if check.Name == "InvalidCycleConfiguration" {
			assert.False(t, check.Ok)
		} else {
			assert.True(t, check.Ok)
		}
	}

<<<<<<< HEAD
	Health(db, s3RW, cmsNotifier, sched, errors.New("something wrong happened"), upService1, upService2)(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck should return 200")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToNativeDatabase","ok":true`, "Database should be healthy")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToS3","ok":true`, "S3 should be healthy")
	assert.Contains(t, string(body), `"name":"CheckCMSNotifierHealth","ok":true`, "CMS notifier should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCycles","ok":true`, "Cycles should be healthy")
	assert.Contains(t, string(body), `"name":"InvalidCycleConfiguration","ok":false`, "Cycles configuration should not be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCluster","ok":true`, "Cluster should be healthy")
	assert.Contains(t, string(body), `"name":"ActivePublishingCluster","ok":true`, "Cluster should not be in failover state")

	sched.AssertExpectations(t)
	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	s3RW.AssertExpectations(t)
	cmsNotifier.AssertExpectations(t)
	upService1.AssertExpectations(t)
	upService2.AssertExpectations(t)
=======
	for _, m := range mocks {
		mock.AssertExpectationsForObjects(t, m)
	}
>>>>>>> master
}

func TestUnhappyClusterHealthcheckWithSchedulerShutdown(t *testing.T) {
	endpoint, mocks := setupTestHealthcheckEndpoint(nil)
	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	upService2 := mocks["service2"].(*cluster.MockService)
	upService2.ExpectedCalls = make([]*mock.Call, 0)

	upService2.On("GTG").Return(errors.New("not good to go"))
	upService2.On("Name").Return("An UPP service")

	sched := mocks["scheduler"].(*scheduler.MockScheduler)
	sched.On("Shutdown").Return(nil)
	sched.On("IsEnabled").Return(true)
	sched.On("IsAutomaticallyDisabled").Return(false)
	sched.On("WasAutomaticallyDisabled").Return(false)

	endpoint(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Healthcheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

	for _, check := range checks {
		if check.Name == "UnhealthyCluster" {
			assert.False(t, check.Ok)
		} else {
			assert.True(t, check.Ok)
		}
	}

	for _, m := range mocks {
		mock.AssertExpectationsForObjects(t, m)
	}
}

func TestSchedulerRestartWhenClusterReturnHealthy(t *testing.T) {
	endpoint, mocks := setupTestHealthcheckEndpoint(nil)
	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

<<<<<<< HEAD
	Health(db, s3RW, cmsNotifier, sched, nil, upService1, upService2)(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck should return 200")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToNativeDatabase","ok":true`, "Database should be healthy")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToS3","ok":true`, "S3 should be healthy")
	assert.Contains(t, string(body), `"name":"CheckCMSNotifierHealth","ok":true`, "CMS notifier should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCycles","ok":true`, "Cycles should be healthy")
	assert.Contains(t, string(body), `"name":"InvalidCycleConfiguration","ok":true`, "Cycles configuration should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCluster","ok":false`, "Cluster should not be healthy")
	assert.Contains(t, string(body), `"name":"ActivePublishingCluster","ok":true`, "Cluster should not be in failover state")

	sched.AssertExpectations(t)
	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	s3RW.AssertExpectations(t)
	cmsNotifier.AssertExpectations(t)
	upService1.AssertExpectations(t)
	upService2.AssertExpectations(t)
}
=======
	sched := mocks["scheduler"].(*scheduler.MockScheduler)
	sched.ExpectedCalls = make([]*mock.Call, 0)
>>>>>>> master

	c1 := mocks["cycle1"].(*scheduler.MockCycle)
	c1.ExpectedCalls = make([]*mock.Call, 0)

	c2 := mocks["cycle2"].(*scheduler.MockCycle)
	c2.ExpectedCalls = make([]*mock.Call, 0)

	c1.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"stopped"}})
	c2.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"stopped"}})

	sched.On("IsRunning").Return(false)
	sched.On("IsEnabled").Return(true)
	sched.On("Cycles").Return(map[string]scheduler.Cycle{
		"c1": c1,
		"c2": c2,
	})
	sched.On("Start").Return(nil).Once()
	sched.On("IsAutomaticallyDisabled").Return(false)
	sched.On("WasAutomaticallyDisabled").Return(false)

	upService1 := new(cluster.MockService)
	upService1.On("GTG").Return(nil)

	upService2 := new(cluster.MockService)
	upService2.On("GTG").Return(nil)

	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	db.On("Open").Return(mockTx, nil)
	mockTx.On("Ping").Return(nil)
	mockTx.On("Close").Return()

	s3RW := new(s3.MockReadWriter)
	s3RW.On("Ping").Return(nil)

	cmsNotifier := new(cms.MockNotifier)
	cmsNotifier.On("GTG").Return(nil)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	Health(db, s3RW, cmsNotifier, sched, nil, upService1, upService2)(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck should return 200")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToNativeDatabase","ok":true`, "Database should be healthy")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToS3","ok":true`, "S3 should be healthy")
	assert.Contains(t, string(body), `"name":"CheckCMSNotifierHealth","ok":true`, "CMS notifier should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCycles","ok":true`, "Cycles should be healthy")
	assert.Contains(t, string(body), `"name":"InvalidCycleConfiguration","ok":true`, "Cycles configuration should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCluster","ok":true`, "Cluster should be healthy")
	assert.Contains(t, string(body), `"name":"ActivePublishingCluster","ok":true`, "Cluster should not be in failover state")

	sched.AssertExpectations(t)
	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	s3RW.AssertExpectations(t)
	cmsNotifier.AssertExpectations(t)
	upService1.AssertExpectations(t)
	upService2.AssertExpectations(t)
}

func TestHappyHealthcheckIfManualToggleIsDisabled(t *testing.T) {
	sched := new(scheduler.MockScheduler)

	c1 := new(scheduler.MockCycle)
	c1.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"running"}})
	c2 := new(scheduler.MockCycle)
	c2.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"running"}})

	mockCycles := map[string]scheduler.Cycle{
		"c1": c1,
		"c2": c2,
	}

	sched.On("Cycles").Return(mockCycles)
	sched.On("IsRunning").Return(true)
	sched.On("IsEnabled").Return(false)
	sched.On("IsAutomaticallyDisabled").Return(false)
	sched.On("WasAutomaticallyDisabled").Return(false)

	upService1 := new(cluster.MockService)
	upService1.On("GTG").Return(nil)
	upService2 := new(cluster.MockService)
	upService2.On("GTG").Return(nil)

	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	db.On("Open").Return(mockTx, nil)
	mockTx.On("Ping").Return(nil)
	mockTx.On("Close").Return()

	s3RW := new(s3.MockReadWriter)
	s3RW.On("Ping").Return(nil)

	cmsNotifier := new(cms.MockNotifier)
	cmsNotifier.On("GTG").Return(nil)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	Health(db, s3RW, cmsNotifier, sched, nil, upService1, upService2)(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck should return 200")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToNativeDatabase","ok":true`, "Database should be healthy")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToS3","ok":true`, "S3 should be healthy")
	assert.Contains(t, string(body), `"name":"CheckCMSNotifierHealth","ok":true`, "CMS notifier should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCycles","ok":true`, "Cycles should be healthy")
	assert.Contains(t, string(body), `"name":"InvalidCycleConfiguration","ok":true`, "Cycles configuration should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCluster","ok":true`, "Cluster should be healthy")
	assert.Contains(t, string(body), `"name":"ActivePublishingCluster","ok":true`, "Cluster should not be in failover state")

	sched.AssertExpectations(t)
	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	s3RW.AssertExpectations(t)
	cmsNotifier.AssertExpectations(t)
	upService1.AssertExpectations(t)
	upService2.AssertExpectations(t)
}

func TestUnhappyHealthcheckBecauseOfCurrentFailover(t *testing.T) {
	sched := new(scheduler.MockScheduler)

	c1 := new(scheduler.MockCycle)
	c1.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"running"}})
	c2 := new(scheduler.MockCycle)
	c2.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"running"}})

	mockCycles := map[string]scheduler.Cycle{
		"c1": c1,
		"c2": c2,
	}

	sched.On("Cycles").Return(mockCycles)
	sched.On("IsRunning").Return(true)
	sched.On("IsAutomaticallyDisabled").Return(true)

<<<<<<< HEAD
	upService1 := new(cluster.MockService)
	upService1.On("GTG").Return(nil)
	upService2 := new(cluster.MockService)
	upService2.On("GTG").Return(nil)

	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	db.On("Open").Return(mockTx, nil)
	mockTx.On("Ping").Return(nil)
	mockTx.On("Close").Return()

	s3RW := new(s3.MockReadWriter)
	s3RW.On("Ping").Return(nil)

	cmsNotifier := new(cms.MockNotifier)
	cmsNotifier.On("GTG").Return(nil)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	Health(db, s3RW, cmsNotifier, sched, nil, upService1, upService2)(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck should return 200")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToNativeDatabase","ok":true`, "Database should be healthy")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToS3","ok":true`, "S3 should be healthy")
	assert.Contains(t, string(body), `"name":"CheckCMSNotifierHealth","ok":true`, "CMS notifier should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCycles","ok":true`, "Cycles should be healthy")
	assert.Contains(t, string(body), `"name":"InvalidCycleConfiguration","ok":true`, "Cycles configuration should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCluster","ok":true`, "Cluster should be healthy")
	assert.Contains(t, string(body), `"name":"ActivePublishingCluster","ok":false`, "Cluster should be in failover state")

	sched.AssertExpectations(t)
	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	s3RW.AssertExpectations(t)
	cmsNotifier.AssertExpectations(t)
	upService1.AssertExpectations(t)
	upService2.AssertExpectations(t)
}

func TestUnhappyHealthcheckBecauseOfNotRestartAfterFailback(t *testing.T) {
	sched := new(scheduler.MockScheduler)

	c1 := new(scheduler.MockCycle)
	c1.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"running"}})
	c2 := new(scheduler.MockCycle)
	c2.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"running"}})

	mockCycles := map[string]scheduler.Cycle{
		"c1": c1,
		"c2": c2,
	}

	sched.On("Cycles").Return(mockCycles)
	sched.On("IsRunning").Return(true)
	sched.On("IsAutomaticallyDisabled").Return(false)
	sched.On("WasAutomaticallyDisabled").Return(true)

	upService1 := new(cluster.MockService)
	upService1.On("GTG").Return(nil)
	upService2 := new(cluster.MockService)
	upService2.On("GTG").Return(nil)

	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	db.On("Open").Return(mockTx, nil)
	mockTx.On("Ping").Return(nil)
	mockTx.On("Close").Return()

	s3RW := new(s3.MockReadWriter)
	s3RW.On("Ping").Return(nil)
=======
	endpoint(w, req)
>>>>>>> master

	assert.Equal(t, http.StatusOK, w.Code, "Healthcheck should return 200")
	checks, err := parseHealthcheck(w.Body.String())
	assert.NoError(t, err)

	for _, check := range checks {
		assert.True(t, check.Ok)
	}

<<<<<<< HEAD
	Health(db, s3RW, cmsNotifier, sched, nil, upService1, upService2)(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck should return 200")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToNativeDatabase","ok":true`, "Database should be healthy")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToS3","ok":true`, "S3 should be healthy")
	assert.Contains(t, string(body), `"name":"CheckCMSNotifierHealth","ok":true`, "CMS notifier should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCycles","ok":true`, "Cycles should be healthy")
	assert.Contains(t, string(body), `"name":"InvalidCycleConfiguration","ok":true`, "Cycles configuration should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCluster","ok":true`, "Cluster should be healthy")
	assert.Contains(t, string(body), `"name":"ActivePublishingCluster","ok":false`, "Cluster should be in failover state")

	sched.AssertExpectations(t)
	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	s3RW.AssertExpectations(t)
	cmsNotifier.AssertExpectations(t)
	upService1.AssertExpectations(t)
	upService2.AssertExpectations(t)
=======
	for _, m := range mocks {
		mock.AssertExpectationsForObjects(t, m)
	}
>>>>>>> master
}

func TestHappyGTG(t *testing.T) {
	endpoint, mocks := setupTestGTGEndpoint(nil)
	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	endpoint(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	for _, m := range mocks {
		mock.AssertExpectationsForObjects(t, m)
	}
}

func TestUnhappyGTG(t *testing.T) {
	endpoint, _ := setupTestGTGEndpoint(errors.New("config err"))
	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	endpoint(w, req)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
