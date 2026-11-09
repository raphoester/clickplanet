package list_due_teasers_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_due_teasers_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

type query struct {
	due []*promov1.Teaser
	err error
}

func (q query) Due(context.Context) ([]*promov1.Teaser, error) {
	return q.due, q.err
}

func list(t *testing.T, q query) (*connect.Response[promov1.ListDueTeasersResponse], error) {
	t.Helper()
	return list_due_teasers_handler.New(q).ListDueTeasers(t.Context(), connect.NewRequest(&promov1.ListDueTeasersRequest{})) //nolint:wrapcheck // the test reads it.
}

func TestTheBattlesDueAreAnswered(t *testing.T) {
	due := []*promov1.Teaser{{BattleId: "entry-7"}}

	answer, err := list(t, query{due: due})

	require.NoError(t, err)
	assert.Equal(t, due, answer.Msg.GetTeasers())
}

func TestABusyPublishingIsUnavailableAndAnyOtherFailureIsTheErrorNets(t *testing.T) {
	_, err := list(t, query{err: &teasers.BusyError{Reason: "restarting"}})
	assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))

	off := connect.NewError(connect.CodeFailedPrecondition, errors.New("publishing is off"))
	_, err = list(t, query{err: off})
	require.ErrorIs(t, err, off)
}
