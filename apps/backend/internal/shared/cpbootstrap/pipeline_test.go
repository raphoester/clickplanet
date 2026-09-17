package cpbootstrap_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

func TestAPipelineStopsItsRunnersInOrder(t *testing.T) {
	var (
		mu      sync.Mutex
		stopped []string
	)
	stopAfter := func(name string, delay time.Duration) cpbootstrap.Runner {
		return runner{name: name, run: func(ctx context.Context) {
			<-ctx.Done()
			time.Sleep(delay)
			mu.Lock()
			defer mu.Unlock()
			stopped = append(stopped, name)
		}}
	}

	pipeline := cpbootstrap.Pipeline(
		stopAfter("subscriber", 30*time.Millisecond),
		stopAfter("store", 0),
	)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		pipeline.Run(ctx)
		close(done)
	}()

	cancel()
	<-done

	assert.Equal(t, []string{"subscriber", "store"}, stopped)
	assert.Equal(t, "subscriber>store", pipeline.Name())
}
