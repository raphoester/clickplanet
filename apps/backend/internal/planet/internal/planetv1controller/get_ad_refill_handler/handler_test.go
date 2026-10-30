package get_ad_refill_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_ad_refill_handler"
)

type stubUseCase time.Time

func (s stubUseCase) Execute(context.Context) (time.Time, error) { return time.Time(s), nil }

func answered(t *testing.T, at time.Time) int64 {
	t.Helper()

	res, err := get_ad_refill_handler.New(stubUseCase(at)).
		GetAdRefill(t.Context(), connect.NewRequest(&planetv1.GetAdRefillRequest{}))
	require.NoError(t, err)
	return res.Msg.GetAvailableAtUnixMs()
}

func TestWhenTheNextAdRefillMayBeIsAnswered(t *testing.T) {
	next := time.Date(2026, 10, 30, 13, 0, 0, 0, time.UTC)

	assert.Equal(t, next.UnixMilli(), answered(t, next))
}

func TestACallerWithNoAccountIsAnsweredZero(t *testing.T) {
	assert.Zero(t, answered(t, time.Time{}))
}
