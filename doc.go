// Package gofuncy provides context-aware, observable goroutine management
// with built-in resilience patterns (retry, circuit breaker, fallback).
//
// It replaces raw go func() calls with structured APIs that propagate context,
// recover from panics, and emit OpenTelemetry metrics and traces.
package gofuncy
