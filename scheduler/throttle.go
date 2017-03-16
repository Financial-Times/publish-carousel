package scheduler

import (
	"context"
	"time"

	log "github.com/Sirupsen/logrus"

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

func NewCappedDynamicThrottle(interval time.Duration, minThrottle time.Duration, maxThrottle time.Duration, publishes int, burst int) (Throttle, context.CancelFunc) {
	rateInterval := determineRateInterval(interval, minThrottle, maxThrottle, publishes)
	ctx, cancel := context.WithCancel(context.Background())
	limiter := rate.NewLimiter(rate.Every(rateInterval), burst)

	throttle := &DefaultThrottle{Context: ctx, Limiter: limiter}
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

func NewDynamicThrottle(minimumThrottle time.Duration, interval time.Duration, publishes int, burst int) (Throttle, context.CancelFunc) {
	publishDelay := time.Duration(interval.Nanoseconds() / int64(publishes))
	if publishDelay < minimumThrottle {
		interval = minimumThrottle
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
