package gofuncy_test

import (
	"context"
	"testing"
	"time"
)

var run = func(ctx context.Context) error {
	time.Sleep(time.Millisecond)
	return nil
}

func BenchmarkGoRaw(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		errChan := make(chan error, 1)

		go func() {
			errChan <- run(b.Context())

			close(errChan)
		}()

		<-errChan
	}
}
