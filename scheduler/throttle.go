package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"golang.org/x/time/rate"
)

type Throttle interface {
	Queue() error
	Stop()
	Interval() time.Duration
}

type DefaultThrottle struct {
	Context  context.Context
	Limiter  *rate.Limiter
	cancel   context.CancelFunc
	interval time.Duration
}

func (d *DefaultThrottle) Queue() error {
	return d.Limiter.Wait(d.Context)
}

func (d *DefaultThrottle) Stop() {
	d.cancel()
}

func (d *DefaultThrottle) Interval() time.Duration {
	return d.interval
}

func (d *DefaultThrottle) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{"interval": d.interval.String()}
	w := bytes.NewBuffer(make([]byte, 0, 1024))
	err := json.NewEncoder(w).Encode(m)

	return w.Bytes(), err
}

func (d *DefaultThrottle) UnmarshalJSON(in []byte) error {
	m := make(map[string]string)
	err := json.NewDecoder(bytes.NewReader(in)).Decode(&m)
	if err == nil {
		if interval, ok := m["interval"]; ok {
			var duration time.Duration
			duration, err = time.ParseDuration(interval)
			if err == nil {
				d.interval = duration
			}
		} else {
			err = fmt.Errorf("no interval value, cannot be unmarshalled to DefaultThrottle")
		}
	}

	return err
}

func NewCappedDynamicThrottle(interval time.Duration, minThrottle time.Duration, maxThrottle time.Duration, publishes int, burst int) (Throttle, context.CancelFunc) {
	rateInterval := determineRateInterval(interval, minThrottle, maxThrottle, publishes)
	ctx, cancel := context.WithCancel(context.Background())
	limiter := rate.NewLimiter(rate.Every(rateInterval), burst)

	throttle := &DefaultThrottle{Context: ctx, Limiter: limiter, interval: rateInterval, cancel: cancel}
	return throttle, cancel
}

func determineRateInterval(interval time.Duration, minThrottle time.Duration, maxThrottle time.Duration, publishes int) time.Duration {
	publishDelay := time.Duration(interval.Nanoseconds() / int64(publishes))
	if publishDelay < minThrottle {
		publishDelay = minThrottle
	} else if publishDelay > maxThrottle {
		publishDelay = maxThrottle
	}

	log.WithField("publishes", publishes).WithField("rate", publishDelay.String()).Info("Determined rate for dynamic throttle.")
	return publishDelay
}

func NewDynamicThrottle(interval time.Duration, minimumThrottle time.Duration, publishes int, burst int) (Throttle, context.CancelFunc) {
	publishDelay := time.Duration(interval.Nanoseconds() / int64(publishes))
	if publishDelay < minimumThrottle {
		interval = minimumThrottle
	}

	ctx, cancel := context.WithCancel(context.Background())
	limiter := rate.NewLimiter(rate.Every(interval), burst)

	throttle := &DefaultThrottle{Context: ctx, Limiter: limiter, interval: interval, cancel: cancel}
	return throttle, cancel
}

func NewThrottle(interval time.Duration, burst int) (Throttle, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	limiter := rate.NewLimiter(rate.Every(interval), burst)
	return &DefaultThrottle{Context: ctx, Limiter: limiter, interval: interval, cancel: cancel}, cancel
}
