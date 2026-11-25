package database

import (
	"context"
	"log"
	"time"
)

// Remote services (Neon, Upstash) can be slow to answer the first connection:
// Neon wakes a suspended database, and TLS/DNS from inside a container can take
// a few seconds. Retry for up to ~45s before giving up.
const (
	connectAttempts = 6
	connectTimeout  = 10 * time.Second
	retryDelay      = 3 * time.Second
)

// pingWithRetry calls ping with a per-attempt timeout until it succeeds or the
// attempts run out, logging each failure.
func pingWithRetry(name string, ping func(ctx context.Context) error) error {
	var err error
	for attempt := 1; attempt <= connectAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
		err = ping(ctx)
		cancel()
		if err == nil {
			if attempt > 1 {
				log.Printf("%s: connected on attempt %d", name, attempt)
			}
			return nil
		}
		log.Printf("%s: attempt %d/%d failed: %v", name, attempt, connectAttempts, err)
		if attempt < connectAttempts {
			time.Sleep(retryDelay)
		}
	}
	return err
}
