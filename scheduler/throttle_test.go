package scheduler

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDynamicThrottle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
		return
	}

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

func TestCappedThrottle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping - this test can take several seconds.")
		return
	}

	throttle, _ := NewCappedDynamicThrottle(time.Minute, time.Millisecond, 100*time.Millisecond, 1, 1)
	start := time.Now()
	throttle.Queue()

	lap1 := time.Now()
	assert.WithinDuration(t, start, lap1, 10*time.Millisecond)

	throttle.Queue()
	lap2 := time.Now()
	assert.WithinDuration(t, lap1.Add(100*time.Millisecond), lap2, 10*time.Millisecond)

	throttle.Queue()
	lap3 := time.Now()
	assert.WithinDuration(t, lap2.Add(100*time.Millisecond), lap3, 10*time.Millisecond)
}

func TestRateInterval(t *testing.T) {
	if testing.Short() {
		t.Skip("long test.")
	}

	interval := time.Minute
	minThrottle := time.Second
	maxThrottle := time.Second * 30

	actual := determineRateInterval(interval, minThrottle, maxThrottle, 10)
	assert.Equal(t, time.Second*6, actual)

	actual = determineRateInterval(interval, minThrottle, maxThrottle, 120)
	assert.Equal(t, time.Second, actual)

	actual = determineRateInterval(interval, minThrottle, maxThrottle, 1)
	assert.Equal(t, time.Second*30, actual)
}

func TestMarshaler(t *testing.T) {
	throttle, _ := NewThrottle(30*time.Second, 1)

	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	err := json.NewEncoder(buf).Encode(throttle)
	assert.NoError(t, err, "marshalling should have occurred without error")
	assert.Equal(t, `{"interval":"30s"}`, strings.Trim(string(buf.Bytes()), "\r\n"), "marshalled JSON")
}

func TestUnmarshaler(t *testing.T) {
	throttle := DefaultThrottle{}
	err := json.NewDecoder(strings.NewReader(`{"interval":"20s"}`)).Decode(&throttle)
	assert.NoError(t, err, "unmarshalling should have occurred without error")
	assert.Equal(t, 20*time.Second, throttle.Interval(), "unmarshalled interval")
}

func TestUnmarshalerErrorUnmatchedJSON(t *testing.T) {
	throttle := DefaultThrottle{}
	err := json.NewDecoder(strings.NewReader(`{"foo":"bar"}`)).Decode(&throttle)
	assert.Error(t, err, "unmarshalling should have failed")
}

func TestUnmarshalerErrorInvalidDuration(t *testing.T) {
	throttle := DefaultThrottle{}
	err := json.NewDecoder(strings.NewReader(`{"interval":"foo"}`)).Decode(&throttle)
	assert.Error(t, err, "unmarshalling should have failed")
}
