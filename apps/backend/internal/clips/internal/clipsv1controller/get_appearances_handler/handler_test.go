package get_appearances_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/get_appearances_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type query struct {
	asked []casts.AccountID
}

func (q *query) Appearances(_ context.Context, account casts.AccountID) (*clipsv1.GetAppearancesResponse, error) {
	q.asked = append(q.asked, account)
	return &clipsv1.GetAppearancesResponse{Appearances: []*clipsv1.Appearance{{PostId: "p1"}}}, nil
}

func TestTheCallerReadsItsOwn(t *testing.T) {
	const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a01"
	read := &query{}

	answer, err := get_appearances_handler.New(read).GetAppearances(cpctx.AddAccountToContext(t.Context(), ada),
		connect.NewRequest(&clipsv1.GetAppearancesRequest{}))
	require.NoError(t, err)

	assert.Equal(t, "p1", answer.Msg.GetAppearances()[0].GetPostId())
	require.Len(t, read.asked, 1)
	assert.Equal(t, ada, read.asked[0].String())
}

func TestNoAccountIsUnauthenticated(t *testing.T) {
	read := &query{}

	_, err := get_appearances_handler.New(read).GetAppearances(t.Context(), connect.NewRequest(&clipsv1.GetAppearancesRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.ErrorIs(t, err, caller.ErrNoAccount)
	assert.Empty(t, read.asked)
}
