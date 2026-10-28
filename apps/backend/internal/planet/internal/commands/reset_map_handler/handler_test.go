package reset_map_handler_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/commands/reset_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
)

type resetting struct {
	asked []resets.Reset
}

func (r *resetting) Execute(_ context.Context, reset resets.Reset) (bool, error) {
	r.asked = append(r.asked, reset)
	return true, nil
}

var at = time.Date(2026, 11, 1, 15, 0, 0, 0, time.UTC)

func TestAResetIsAppliedAtItsTime(t *testing.T) {
	asked := &resetting{}

	require.NoError(t, reset_map_handler.New(asked).Handle(t.Context(),
		&planetv1.ResetMap{ResetId: "season-1", DecidedAt: timestamppb.New(at)}))

	assert.Equal(t, []resets.Reset{{ID: "season-1", At: at}}, asked.asked)
}

func TestAResetWithNoIDOrNoTimeIsDropped(t *testing.T) {
	asked := &resetting{}
	handler := reset_map_handler.New(asked)

	require.NoError(t, handler.Handle(t.Context(), &planetv1.ResetMap{DecidedAt: timestamppb.New(at)}))
	require.NoError(t, handler.Handle(t.Context(), &planetv1.ResetMap{ResetId: "season-1"}))

	assert.Empty(t, asked.asked)
}
