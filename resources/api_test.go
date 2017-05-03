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
	api := API([]byte(`host: localhost`))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://129.1.1.160/__api", nil)

	api(w, r)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `host: 129.1.1.160`, strings.TrimSpace(w.Body.String()))
	assert.Equal(t, "text/vnd.yaml", w.Header().Get("Content-Type"))
}

func TestAPIEndpointYAMLFails(t *testing.T) {
	api := API([]byte(`hi i am an api`))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://129.1.1.160/__api", nil)

	api(w, r)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `hi i am an api`, strings.TrimSpace(w.Body.String()))
	assert.Equal(t, "text/vnd.yaml", w.Header().Get("Content-Type"))
}

func TestAPIEndpointURLFails(t *testing.T) {
	api := API([]byte(`host: localhost`))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://192.168.1.1/__api", nil)

	api(w, r)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, `host: 192.168.1.1`, strings.TrimSpace(w.Body.String()))
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
