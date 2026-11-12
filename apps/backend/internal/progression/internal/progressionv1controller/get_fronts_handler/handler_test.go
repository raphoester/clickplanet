package get_fronts_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/get_fronts_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

type stubQuery struct {
	answer *progressionv1.GetFrontsResponse
	err    error
	asked  []accounts.AccountID
}

func (s *stubQuery) Fronts(_ context.Context, account accounts.AccountID) (*progressionv1.GetFrontsResponse, error) {
	s.asked = append(s.asked, account)
	return s.answer, s.err
}

func TestTheCallersAnswerIsTheQuerys(t *testing.T) {
	query := &stubQuery{answer: &progressionv1.GetFrontsResponse{PlaysFor: []*progressionv1.CountryTiles{{CountryId: "fr", Tiles: 4}}}}

	res, err := get_fronts_handler.New(query).GetFronts(cpctx.AddAccountToContext(t.Context(), ada), connect.NewRequest(&progressionv1.GetFrontsRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(query.answer, res.Msg))
	account, err := accounts.AccountIDOf(ada)
	require.NoError(t, err)
	assert.Equal(t, []accounts.AccountID{account}, query.asked)
}

func TestACallerWithNoAccountIsUnauthenticatedAndReadsNothing(t *testing.T) {
	query := &stubQuery{}

	_, err := get_fronts_handler.New(query).GetFronts(t.Context(), connect.NewRequest(&progressionv1.GetFrontsRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, query.asked)
}
