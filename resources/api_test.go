package resources

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIEndpoint(t *testing.T) {
	api := API([]byte(`hi - i am an api`))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/__api", nil)

	api(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `hi - i am an api`, w.Body.String())
	assert.Equal(t, "text/vnd.yaml", w.Header().Get("Content-Type"))
}

func TestAPIFailed(t *testing.T) {
	apiYml, _ := ioutil.ReadFile("./a-file-that-doesnt-exist.yml")
	api := API(apiYml)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/__api", nil)

	api(w, r)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, ``, w.Body.String())
}

func TestUpdateLogLevelInfo(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/__log", strings.NewReader(`{"level":"info"}`))
	LogLevel(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `Updated log level to "info"`, w.Body.String())
}

func TestUpdateLogLevelDebug(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/__log", strings.NewReader(`{"level":"debug"}`))
	LogLevel(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `Updated log level to "debug"`, w.Body.String())
}

func TestUpdateLogLevelInvalidLevel(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/__log", strings.NewReader(`{"level":"warn"}`))
	LogLevel(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, `Invalid level. Please select one of "debug" or "info"`, strings.TrimSpace(w.Body.String()))
}

func TestUpdateLogLevelInvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/__log", strings.NewReader(``))
	LogLevel(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, `Failed to parse log level update request`, strings.TrimSpace(w.Body.String()))
}
