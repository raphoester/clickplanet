package cpbootstrap

import (
	"context"
	"strings"
	"sync"
)

// Pipeline runs every runner at once, and stops them in the order given: each one's context is cancelled only
// once the one before it has returned. A subscriber that feeds a store goes first, so the store's last
// flush holds every event the subscriber drained.
func Pipeline(runners ...Runner) Runner {
	return pipeline{runners: runners}
}

type pipeline struct {
	runners []Runner
}

func (p pipeline) Name() string {
	names := make([]string, len(p.runners))
	for i, runner := range p.runners {
		names[i] = runner.Name()
	}
	return strings.Join(names, ">")
}

func (p pipeline) Run(ctx context.Context) {
	var wg sync.WaitGroup
	stopped := ctx

	for _, runner := range p.runners {
		runnerCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		context.AfterFunc(stopped, cancel)

		done, finished := context.WithCancel(context.Background())
		wg.Go(func() {
			defer finished()
			runner.Run(runnerCtx)
		})
		stopped = done
	}

	wg.Wait()
}
