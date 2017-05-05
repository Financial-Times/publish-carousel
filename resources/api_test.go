package resources

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIEndpoint(t *testing.T) {
	api := API([]byte(`hi - i am an api`))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/__api", nil)

	api(w, r)

	assert.Equal(t, 200, w.Code)
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
