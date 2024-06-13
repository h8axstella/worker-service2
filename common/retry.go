package common

import (
	"time"
)

const (
	MaxConcurrentRequests = 500
	MaxRetryAttempts      = 3
)

func Retry(attempts int, sleep int, f func() error) error {
	if err := f(); err != nil {
		if attempts--; attempts > 0 {
			time.Sleep(time.Duration(sleep) * time.Second)
			return Retry(attempts, sleep, f)
		}
		return err
	}
	return nil
}
