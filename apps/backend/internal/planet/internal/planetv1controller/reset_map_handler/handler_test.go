package reset_map_handler_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/reset_map_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
)

// resetting stops on a cancelled context, as a wipe in postgres does.
type resetting struct {
	applied []resets.Reset
	err     error
}

func (r *resetting) Execute(ctx context.Context, reset resets.Reset) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err //nolint:wrapcheck // the context names it.
	}
	if r.err != nil {
		return false, r.err
	}
	if slices.Contains(r.applied, reset) {
		return false, nil
	}
	r.applied = append(r.applied, reset)
	return true, nil
}

var at = time.Date(2026, 11, 1, 15, 0, 0, 0, time.UTC)

func reset(ctx context.Context, useCase *resetting, req *planetv1.ResetMapRequest) error {
	_, err := reset_map_handler.New(useCase).ResetMap(ctx, connect.NewRequest(req))
	return err //nolint:wrapcheck // the handler's answer, as the caller reads it.
}

func TestAResetIsAppliedAtItsTimeOnce(t *testing.T) {
	useCase := &resetting{}
	req := &planetv1.ResetMapRequest{ResetId: "season-1", DecidedAt: timestamppb.New(at)}

	require.NoError(t, reset(t.Context(), useCase, req))
	require.NoError(t, reset(t.Context(), useCase, req), "a reset applied before answers the same")

	assert.Equal(t, []resets.Reset{{ID: "season-1", At: at}}, useCase.applied)
}

func TestAResetWithNoIDOrNoTimeIsAnInvalidArgument(t *testing.T) {
	useCase := &resetting{}

	for name, req := range map[string]*planetv1.ResetMapRequest{
		"no id":   {DecidedAt: timestamppb.New(at)},
		"no time": {ResetId: "season-1"},
	} {
		err := reset(t.Context(), useCase, req)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), name)
		require.ErrorIs(t, err, resets.ErrInvalidReset, name)
	}
	assert.Empty(t, useCase.applied)
}

func TestTheWipeGoesOnWhenTheCallerGivesUp(t *testing.T) {
	useCase := &resetting{}
	gaveUp, cancel := context.WithCancel(t.Context())
	cancel()

	require.NoError(t, reset(gaveUp, useCase, &planetv1.ResetMapRequest{ResetId: "season-1", DecidedAt: timestamppb.New(at)}))

	assert.Equal(t, []resets.Reset{{ID: "season-1", At: at}}, useCase.applied)
}

func TestAFailureIsPassedOnForTheErrorNet(t *testing.T) {
	down := errors.New("postgres is down")

	err := reset(t.Context(), &resetting{err: down}, &planetv1.ResetMapRequest{ResetId: "season-1", DecidedAt: timestamppb.New(at)})

	require.ErrorIs(t, err, down)
}
