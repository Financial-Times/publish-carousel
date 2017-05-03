package resources

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIEndpointHTTP(t *testing.T) {
	api := API([]byte("host: localhost"))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://129.1.1.160/__publish-carousel/__api", nil)

	api(w, r)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "basePath: /__publish-carousel\nhost: 129.1.1.160\nschemes:\n- http", strings.TrimSpace(w.Body.String()))
	assert.Equal(t, "text/vnd.yaml", w.Header().Get("Content-Type"))
}

func TestAPIEndpointHTTPS(t *testing.T) {
	api := API([]byte(`host: localhost`))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://129.1.1.160/__api", nil)

	api(w, r)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "basePath: /\nhost: 129.1.1.160\nschemes:\n- https", strings.TrimSpace(w.Body.String()))
	assert.Equal(t, "text/vnd.yaml", w.Header().Get("Content-Type"))
}

func TestAPIEndpointNoScheme(t *testing.T) {
	api := API([]byte("host: localhost"))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/__publish-carousel/__api", nil)

	api(w, r)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "basePath: /__publish-carousel\nhost: example.com\nschemes:\n- http\n- https", strings.TrimSpace(w.Body.String()))
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

func TestAPIFailed(t *testing.T) {
	apiYml, _ := ioutil.ReadFile("./a-file-that-doesnt-exist.yml")
	api := API(apiYml)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/__api", nil)

	api(w, r)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, ``, w.Body.String())
}
