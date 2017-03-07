package scheduler

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

type Throttle interface {
	Queue() error
	Stop()
}

type DefaultThrottle struct {
	Context context.Context
	Limiter *rate.Limiter
	cancel  context.CancelFunc
}

func (d *DefaultThrottle) Queue() error {
	return d.Limiter.Wait(d.Context)
}

func (d *DefaultThrottle) Stop() {
	d.cancel()
}

func NewDynamicThrottle(interval time.Duration, publishes int, burst int) (Throttle, context.CancelFunc) {
	pubishDelay := time.Duration(interval.Nanoseconds() / int64(publishes))
	if pubishDelay < time.Second {
		interval = time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	limiter := rate.NewLimiter(rate.Every(interval), burst)

	throttle := &DefaultThrottle{Context: ctx, Limiter: limiter}
	return throttle, cancel
}

func NewThrottle(interval time.Duration, burst int) (Throttle, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	limiter := rate.NewLimiter(rate.Every(interval), burst)
	return &DefaultThrottle{Context: ctx, Limiter: limiter}, cancel
}
