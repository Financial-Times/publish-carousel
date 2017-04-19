package resources

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/cycles/1234id", nil)
	w := httptest.NewRecorder()

	MethodNotAllowed()(w, req)
	assert.Equal(t, 405, w.Code)
}
