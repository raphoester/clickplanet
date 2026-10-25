package enforce_rules_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/enforce_rules_usecase"
)

type counted struct{ passes chan struct{} }

func (c counted) Execute(context.Context) (rules.Snapshot, error) {
	c.passes <- struct{}{}
	return rules.Snapshot{}, nil
}

func TestTheRunnerEnforcesOnEachTickUntilStopped(t *testing.T) {
	executor := counted{passes: make(chan struct{}, 16)}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		enforce_rules_usecase.NewRunner(10*time.Millisecond, executor).Run(ctx)
		close(done)
	}()

	for range 3 {
		<-executor.passes
	}
	cancel()
	<-done

	assert.Equal(t, "planet-rules", enforce_rules_usecase.NewRunner(time.Second, executor).Name())
}
