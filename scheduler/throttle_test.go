package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDynamicThrottle(t *testing.T) {
	throttle, _ := NewDynamicThrottle(1*time.Second, 1*time.Second, 1, 1)
	start := time.Now()
	throttle.Queue()

	lap1 := time.Now()
	assert.WithinDuration(t, start, lap1, 10*time.Millisecond)

	throttle.Queue()
	lap2 := time.Now()
	assert.WithinDuration(t, lap1.Add(1*time.Second), lap2, 10*time.Millisecond)

	throttle.Queue()
	lap3 := time.Now()
	assert.WithinDuration(t, lap2.Add(1*time.Second), lap3, 10*time.Millisecond)
}
