package native

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	sha, err := Hash([]byte(`i am a payload`))
	t.Log(sha)
	assert.NoError(t, err)
	assert.Equal(t, "7a8cd3bb0191504451c9b840d20accda5e86a090b2f7a24eaadffeb6", sha)
}
