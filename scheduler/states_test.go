package scheduler

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatesMarshalJSON(t *testing.T) {
	s := State{lock: &sync.RWMutex{}, states: []string{"hey", "you"}}
	b, err := json.Marshal(&s)

	assert.NoError(t, err)
	assert.Equal(t, `["hey","you"]`, string(b))
}

func TestStatesUnmarshalJSON(t *testing.T) {
	s := State{lock: &sync.RWMutex{}}
	err := json.Unmarshal([]byte(`["hey","you"]`), &s)

	assert.NoError(t, err)
	assert.Equal(t, s.states, []string{"hey", "you"})
	assert.Equal(t, s.stateSet, map[string]struct{}{"hey": struct{}{}, "you": struct{}{}})
}
