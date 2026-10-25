package tell_seasons_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/tell_seasons_usecase"
)

type recordingExecutor struct {
	asked chan []calendar.Number
}

func (r recordingExecutor) Execute(_ context.Context, told []calendar.Number) ([]tell_seasons_usecase.Told, error) {
	r.asked <- told
	if len(told) == 0 {
		return []tell_seasons_usecase.Told{{Season: 0, Players: 2}}, nil
	}
	return nil, nil
}

func TestTheRunnerTellsAtStartThenEveryIntervalAndRemembersWhatItTold(t *testing.T) {
	executor := recordingExecutor{asked: make(chan []calendar.Number, 10)}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})

	go func() {
		tell_seasons_usecase.NewRunner(10*time.Millisecond, executor).Run(ctx)
		close(done)
	}()

	assert.Empty(t, <-executor.asked, "a process starts having told nothing")
	assert.Equal(t, []calendar.Number{0}, <-executor.asked)
	cancel()
	assert.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
}
