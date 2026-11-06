package list_renames_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/list_renames_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/list_renames_handler/renames_query"
)

type stubQuery struct {
	asked *renames_query.Filter
}

func (s *stubQuery) Renames(_ context.Context, filter renames_query.Filter) (*playerv1.ListRenamesResponse, error) {
	s.asked = &filter
	return &playerv1.ListRenamesResponse{}, nil
}

const adaID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

func list(t *testing.T, query *stubQuery, req *playerv1.ListRenamesRequest) (*connect.Response[playerv1.ListRenamesResponse], error) {
	t.Helper()

	return list_renames_handler.New(query).ListRenames(t.Context(), connect.NewRequest(req)) //nolint:wrapcheck // the tests read the code.
}

func TestTheFilterReachesTheQueryAndTheAnswerIsNeverCached(t *testing.T) {
	since := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	query := &stubQuery{}
	ada, err := players.AccountIDOf(adaID)
	require.NoError(t, err)

	res, err := list(t, query, &playerv1.ListRenamesRequest{AccountId: adaID, Since: timestamppb.New(since), Limit: 7})
	require.NoError(t, err)

	assert.Equal(t, &renames_query.Filter{Account: ada, Since: since, Limit: 7}, query.asked)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestNoAccountIsEveryAccountAndNoTimeEveryRenameKept(t *testing.T) {
	query := &stubQuery{}

	_, err := list(t, query, &playerv1.ListRenamesRequest{})
	require.NoError(t, err)

	assert.Equal(t, players.AccountID{}, query.asked.Account)
	assert.Equal(t, time.Unix(0, 0), query.asked.Since)
}

func TestAnAccountThatIsNotAUUIDIsInvalidAndNothingIsRead(t *testing.T) {
	query := &stubQuery{}

	_, err := list(t, query, &playerv1.ListRenamesRequest{AccountId: "Ada"})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Nil(t, query.asked)
}
