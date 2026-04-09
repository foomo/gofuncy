package gofuncy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"github.com/foomo/gofuncy/semconv/gofuncyconv"
)

// ErrCircuitOpen is returned when the circuit breaker is open and not
// accepting requests.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerOption configures circuit breaker behavior.
type CircuitBreakerOption func(*circuitBreakerConfig)

type circuitBreakerConfig struct {
	threshold     int
	cooldown      time.Duration
	failureIf     func(error) bool
	onStateChange func(from, to CircuitState)
}

// CircuitState represents the current state of a circuit breaker.
type CircuitState int

const (
	// CircuitClosed is the normal operating state where requests pass through.
	CircuitClosed CircuitState = iota
	// CircuitOpen is the state where requests are rejected immediately.
	CircuitOpen
	// CircuitHalfOpen is the state where a single probe request is allowed.
	CircuitHalfOpen
)

// String implements fmt.Stringer for CircuitState.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// CircuitBreaker holds the state for a circuit breaker instance. It is safe for
// concurrent use and should be shared across all calls to the same dependency.
type CircuitBreaker struct {
	mu           sync.Mutex
	state        CircuitState
	failures     int
	lastFailedAt time.Time
	cfg          circuitBreakerConfig
}

// NewCircuitBreaker creates a new CircuitBreaker with the given options.
func NewCircuitBreaker(opts ...CircuitBreakerOption) *CircuitBreaker {
	cb := &CircuitBreaker{
		cfg: circuitBreakerConfig{
			threshold: 5,
			cooldown:  30 * time.Second,
			failureIf: defaultCircuitBreakerFailureIf,
		},
	}

	for _, opt := range opts {
		opt(&cb.cfg)
	}

	return cb
}

// middleware returns a Middleware that implements the circuit breaker pattern.
func (cb *CircuitBreaker) middleware(m metric.Meter, name string) Middleware {
	rejected, err := gofuncyconv.NewGoroutinesRejected(m)
	if err != nil {
		otel.Handle(err)
	}

	return func(fn Func) Func {
		return func(ctx context.Context) error {
			cb.mu.Lock()

			switch cb.state {
			case CircuitOpen:
				if time.Since(cb.lastFailedAt) < cb.cfg.cooldown {
					cb.mu.Unlock()

					rejected.Add(ctx, 1, name)

					return ErrCircuitOpen
				}
				// Cooldown elapsed — transition to half-open for a probe
				notify, from := cb.transition(CircuitHalfOpen)
				cb.mu.Unlock()

				if notify != nil {
					notify(from, CircuitHalfOpen)
				}

			case CircuitHalfOpen:
				// Another probe is already in flight; reject
				cb.mu.Unlock()

				rejected.Add(ctx, 1, name)

				return ErrCircuitOpen

			default: // CircuitClosed
				cb.mu.Unlock()
			}

			err := fn(ctx)

			cb.mu.Lock()

			var (
				notify       func(CircuitState, CircuitState)
				from         CircuitState
				transitionTo CircuitState
			)

			if err != nil && cb.cfg.failureIf(err) {
				cb.failures++
				cb.lastFailedAt = time.Now()

				if cb.failures >= cb.cfg.threshold {
					notify, from = cb.transition(CircuitOpen)
					transitionTo = CircuitOpen
				}

				cb.mu.Unlock()

				if notify != nil {
					notify(from, transitionTo)
				}

				return err
			}

			// Success — reset
			cb.failures = 0

			if cb.state == CircuitHalfOpen {
				notify, from = cb.transition(CircuitClosed)
				transitionTo = CircuitClosed
			}

			cb.mu.Unlock()

			if notify != nil {
				notify(from, transitionTo)
			}

			return err
		}
	}
}

// transition sets the new state and returns the callback to invoke after
// the mutex is released (if any). Must be called while cb.mu is held.
func (cb *CircuitBreaker) transition(to CircuitState) (func(CircuitState, CircuitState), CircuitState) {
	prev := cb.state
	cb.state = to

	if cb.cfg.onStateChange != nil && prev != to {
		return cb.cfg.onStateChange, prev
	}

	return nil, prev
}

// CircuitBreakerThreshold sets the number of consecutive failures before the
// circuit opens. Defaults to 5.
func CircuitBreakerThreshold(n int) CircuitBreakerOption {
	return func(c *circuitBreakerConfig) {
		c.threshold = n
	}
}

// CircuitBreakerCooldown sets the duration the circuit stays open before
// allowing a probe request. Defaults to 30s.
func CircuitBreakerCooldown(d time.Duration) CircuitBreakerOption {
	return func(c *circuitBreakerConfig) {
		c.cooldown = d
	}
}

// CircuitBreakerIf sets a custom function to determine whether an error counts
// as a failure. By default, all errors except context errors and panics count.
func CircuitBreakerIf(fn func(error) bool) CircuitBreakerOption {
	return func(c *circuitBreakerConfig) {
		c.failureIf = fn
	}
}

// CircuitBreakerOnStateChange sets a callback invoked when the circuit
// transitions between states.
func CircuitBreakerOnStateChange(fn func(from, to CircuitState)) CircuitBreakerOption {
	return func(c *circuitBreakerConfig) {
		c.onStateChange = fn
	}
}

func defaultCircuitBreakerFailureIf(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var panicErr *PanicError

	return !errors.As(err, &panicErr)
}
