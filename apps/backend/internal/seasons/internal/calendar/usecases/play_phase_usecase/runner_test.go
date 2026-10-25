package play_phase_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/play_phase_usecase"
)

type counted struct{ passes chan struct{} }

func (c counted) Execute(context.Context) (calendar.Phase, error) {
	c.passes <- struct{}{}
	return calendar.Phase{}, nil
}

func TestTheRunnerPlaysAtOnceThenOnEachTickUntilStopped(t *testing.T) {
	executor := counted{passes: make(chan struct{}, 16)}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		play_phase_usecase.NewRunner(10*time.Millisecond, executor).Run(ctx)
		close(done)
	}()

	for range 3 {
		<-executor.passes
	}
	cancel()
	<-done

	assert.Equal(t, "seasons-phases", play_phase_usecase.NewRunner(time.Second, executor).Name())
}
