package circuitbreaker

import (
	"time"

	"github.com/sony/gobreaker"
)

// NewCB creates a new CircuitBreaker with sensible defaults for gRPC calls.
func NewCB(name string) *gobreaker.CircuitBreaker {
	st := gobreaker.Settings{
		Name:          name,
		MaxRequests:   3,                // maximum number of requests allowed to pass through when the CircuitBreaker is half-open
		Interval:      10 * time.Second, // cyclic period of the closed state
		Timeout:       5 * time.Second,  // period of the open state, after which it becomes half-open
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip if there are at least 5 failures and the failure ratio is >= 50%
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.5
		},
	}
	return gobreaker.NewCircuitBreaker(st)
}
