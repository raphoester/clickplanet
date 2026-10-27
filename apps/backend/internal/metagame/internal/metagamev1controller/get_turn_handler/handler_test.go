package get_turn_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/get_turn_handler"
)

type stubQuery struct {
	turn *metagamev1.Turn
	err  error
}

func (s stubQuery) Turn(context.Context) (*metagamev1.Turn, error) {
	return s.turn, s.err
}

func TestTheTurnIsTheQuerys(t *testing.T) {
	turn := &metagamev1.Turn{Kind: metagamev1.EntryKind_ENTRY_KIND_BATTLE, Description: "Triple points."}

	res, err := get_turn_handler.New(stubQuery{turn: turn}).
		GetTurn(t.Context(), connect.NewRequest(&metagamev1.GetTurnRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(&metagamev1.GetTurnResponse{Turn: turn}, res.Msg))
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := get_turn_handler.New(stubQuery{err: down}).
		GetTurn(t.Context(), connect.NewRequest(&metagamev1.GetTurnRequest{}))

	require.ErrorIs(t, err, down)
}
