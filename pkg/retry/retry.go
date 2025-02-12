package retry

import (
	"github.com/cenkalti/backoff/v4"
	"time"
)

type Config struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	Timeout         time.Duration
	MaxRetries      uint64
}

func (cfg Config) ToBackOff() backoff.BackOff {
	var bo backoff.BackOff = backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(cfg.InitialInterval),
		backoff.WithMaxInterval(cfg.MaxInterval),
		backoff.WithMultiplier(cfg.Multiplier),
	)
	return backoff.WithMaxRetries(bo, cfg.MaxRetries)
}

func Do(op func() error, cfg Config) error {
	return backoff.Retry(op, cfg.ToBackOff())
}
