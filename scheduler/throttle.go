package scheduler

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

type Throttle interface {
	Queue() error
}

type DynamicThrottle struct {
	Context context.Context
	Limiter *rate.Limiter
}

func (d *DynamicThrottle) Queue() error {
	return d.Limiter.Wait(d.Context)
}

func NewDynamicThrottle(interval time.Duration, burst int) (Throttle, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	limiter := rate.NewLimiter(rate.Every(interval), burst)
	return &DynamicThrottle{Context: ctx, Limiter: limiter}, cancel
}
