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

func NewCappedDynamicThrottle(interval time.Duration, cap time.Duration, publishes int, burst int) (Throttle, context.CancelFunc) {
	publishDelay := time.Duration(interval.Nanoseconds() / int64(publishes))
	if publishDelay < time.Second {
		interval = time.Second
	} else if publishDelay > cap {
		interval = cap
	}

	ctx, cancel := context.WithCancel(context.Background())
	limiter := rate.NewLimiter(rate.Every(interval), burst)

	throttle := &DefaultThrottle{Context: ctx, Limiter: limiter}
	return throttle, cancel
}

func NewDynamicThrottle(interval time.Duration, publishes int, burst int) (Throttle, context.CancelFunc) {
	publishDelay := time.Duration(interval.Nanoseconds() / int64(publishes))
	if publishDelay < time.Second {
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
