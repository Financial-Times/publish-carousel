package resources

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/publish-carousel/cluster"
	"github.com/Financial-Times/publish-carousel/cms"
	"github.com/Financial-Times/publish-carousel/native"
	"github.com/Financial-Times/publish-carousel/s3"
	"github.com/Financial-Times/publish-carousel/scheduler"
	"github.com/stretchr/testify/assert"
)

func TestHappyHealthcheck(t *testing.T) {
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

func TestUnhappyMongoDBHealthcheck(t *testing.T) {
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

	upService1 := new(cluster.MockService)
	upService1.On("GTG").Return(nil)
	upService2 := new(cluster.MockService)
	upService2.On("GTG").Return(nil)

	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	db.On("Open").Return(mockTx, errors.New("no MongoDB connectivity"))

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
	assert.Contains(t, string(body), `"name":"CheckConnectivityToNativeDatabase","ok":false`, "Database should not be healthy")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToS3","ok":true`, "S3 should be healthy")
	assert.Contains(t, string(body), `"name":"CheckCMSNotifierHealth","ok":true`, "CMS notifier should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCycles","ok":true`, "Cycles should be healthy")
	assert.Contains(t, string(body), `"name":"InvalidCycleConfiguration","ok":true`, "Cycles configuration should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCluster","ok":true`, "Cluster should be healthy")

	sched.AssertExpectations(t)
	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
	db.AssertExpectations(t)
	s3RW.AssertExpectations(t)
	cmsNotifier.AssertExpectations(t)
	upService1.AssertExpectations(t)
	upService2.AssertExpectations(t)
}

func TestUnhappyS3Healthcheck(t *testing.T) {
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

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	Health(db, s3RW, cmsNotifier, sched, nil, upService1, upService2)(w, req)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Healthcheck should return 200")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToNativeDatabase","ok":true`, "Database should be healthy")
	assert.Contains(t, string(body), `"name":"CheckConnectivityToS3","ok":false`, "S3 should not be healthy")
	assert.Contains(t, string(body), `"name":"CheckCMSNotifierHealth","ok":true`, "CMS notifier should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCycles","ok":true`, "Cycles should be healthy")
	assert.Contains(t, string(body), `"name":"InvalidCycleConfiguration","ok":true`, "Cycles configuration should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCluster","ok":true`, "Cluster should be healthy")

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

func TestUnhappyCMSNotifierHealthcheck(t *testing.T) {
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
	cmsNotifier.On("GTG").Return(errors.New("not available"))

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

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

func TestUnhappyCyclesHealthcheck(t *testing.T) {
	sched := new(scheduler.MockScheduler)

	c1 := new(scheduler.MockCycle)
	c1.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"unhealthy"}})
	c1.On("ID").Return("c1")
	c2 := new(scheduler.MockCycle)
	c2.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"running"}})

	mockCycles := map[string]scheduler.Cycle{
		"c1": c1,
		"c2": c2,
	}

	sched.On("Cycles").Return(mockCycles)
	sched.On("IsRunning").Return(true)

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
	assert.Contains(t, string(body), `"name":"UnhealthyCycles","ok":false`, "Cycles should not be healthy")
	assert.Contains(t, string(body), `"name":"InvalidCycleConfiguration","ok":true`, "Cycles configuration should be healthy")
	assert.Contains(t, string(body), `"name":"UnhealthyCluster","ok":true`, "Cluster should be healthy")

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

func TestUnhappyCycleConfigHealthcheck(t *testing.T) {
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

func TestUnhappyClusterHealthcheckWithSchedulerShutdown(t *testing.T) {
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
	sched.On("Shutdown").Return(nil)

	upService1 := new(cluster.MockService)
	upService1.On("GTG").Return(nil)
	upService2 := new(cluster.MockService)
	upService2.On("GTG").Return(errors.New("not good to go"))
	upService2.On("Name").Return("a UPP service")

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
	assert.Contains(t, string(body), `"name":"UnhealthyCluster","ok":false`, "Cluster should not be healthy")

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

func TestSchedulerRestartWhenClusterReturnHealthy(t *testing.T) {
	sched := new(scheduler.MockScheduler)

	c1 := new(scheduler.MockCycle)
	c1.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"stopped"}})

	c2 := new(scheduler.MockCycle)
	c2.On("Metadata").Return(scheduler.CycleMetadata{State: []string{"stopped"}})

	mockCycles := map[string]scheduler.Cycle{
		"c1": c1,
		"c2": c2,
	}

	sched.On("Cycles").Return(mockCycles)
	sched.On("IsRunning").Return(false)
	sched.On("IsEnabled").Return(true)
	sched.On("Start").Return(nil).Once()

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

func TestHappyGTG(t *testing.T) {
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

	GTG(db, s3RW, cmsNotifier, sched, nil, upService1, upService2)(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "GTG should return 200")

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

func TestUnhappyGTG(t *testing.T) {
	sched := new(scheduler.MockScheduler)

	upService1 := new(cluster.MockService)
	upService2 := new(cluster.MockService)

	db := new(native.MockDB)
	mockTx := new(native.MockTX)
	db.On("Open").Return(mockTx, nil)
	mockTx.On("Ping").Return(nil)
	mockTx.On("Close").Return()

	s3RW := new(s3.MockReadWriter)
	s3RW.On("Ping").Return(errors.New("not a good pong"))

	cmsNotifier := new(cms.MockNotifier)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	GTG(db, s3RW, cmsNotifier, sched, nil, upService1, upService2)(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "GTG should return 503")

	sched.AssertExpectations(t)
	db.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	s3RW.AssertExpectations(t)
	cmsNotifier.AssertExpectations(t)
	upService1.AssertExpectations(t)
	upService2.AssertExpectations(t)
}
