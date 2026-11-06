package get_nameplates_handler_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_nameplates_handler"
)

const (
	adaID   = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	guestID = "5e0c1b2a-3d4e-4f60-8a71-9b2c3d4e5f60"
)

type stubQuery struct {
	answer *playerv1.GetPlayersResponse
	err    error
	asked  [][]players.AccountID
}

func (s *stubQuery) Players(_ context.Context, accounts []players.AccountID) (*playerv1.GetPlayersResponse, error) {
	s.asked = append(s.asked, accounts)
	return s.answer, s.err
}

func getNameplates(t *testing.T, query *stubQuery, ids ...string) (*connect.Response[playerv1.GetNameplatesResponse], error) {
	t.Helper()

	return get_nameplates_handler.New(query).GetNameplates(t.Context(), connect.NewRequest(&playerv1.GetNameplatesRequest{AccountIds: ids})) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestEachNameplateIsThePlayerGetPlayersAnswers(t *testing.T) {
	query := &stubQuery{answer: &playerv1.GetPlayersResponse{Players: []*playerv1.Player{
		{AccountId: adaID, Name: "Ada_L", Admin: true, Color: playerv1.NameColor_NAME_COLOR_ORANGE},
		{AccountId: guestID, Name: "guest_a1b2c3", Guest: true},
	}}}

	res, err := getNameplates(t, query, adaID, guestID)

	require.NoError(t, err)
	assert.True(t, proto.Equal(&playerv1.GetNameplatesResponse{Nameplates: []*playerv1.Nameplate{
		{AccountId: adaID, Name: "Ada_L", Admin: true, Color: playerv1.NameColor_NAME_COLOR_ORANGE},
		{AccountId: guestID, Name: "guest_a1b2c3", Guest: true},
	}}, res.Msg), res.Msg.String())
	ada, err := players.AccountIDOf(adaID)
	require.NoError(t, err)
	guest, err := players.AccountIDOf(guestID)
	require.NoError(t, err)
	assert.Equal(t, [][]players.AccountID{{ada, guest}}, query.asked)
}

func TestAnAccountThePlayersLeaveOutHasNoNameplate(t *testing.T) {
	res, err := getNameplates(t, &stubQuery{answer: &playerv1.GetPlayersResponse{}}, adaID)

	require.NoError(t, err)
	assert.Empty(t, res.Msg.GetNameplates())
}

func TestAPageOfAHundredIsReadAndMoreIsRefused(t *testing.T) {
	page := strings.Split(strings.Repeat(adaID+",", get_nameplates_handler.MaxAsked), ",")[:get_nameplates_handler.MaxAsked]

	query := &stubQuery{answer: &playerv1.GetPlayersResponse{}}
	_, err := getNameplates(t, query, page...)
	require.NoError(t, err)
	assert.Len(t, query.asked, 1)

	query = &stubQuery{answer: &playerv1.GetPlayersResponse{}}
	_, err = getNameplates(t, query, append(page, guestID)...)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.ErrorIs(t, err, get_nameplates_handler.ErrTooMany)
	assert.Empty(t, query.asked)
}

func TestAnIdThatIsNotAnAccountIsRefusedAndReadsNothing(t *testing.T) {
	for _, id := range []string{"", "not-an-account", "00000000-0000-0000-0000-000000000000"} {
		query := &stubQuery{answer: &playerv1.GetPlayersResponse{}}

		_, err := getNameplates(t, query, adaID, id)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
		assert.Empty(t, query.asked, id)
	}
}

func TestTheAnswerIsNeverStored(t *testing.T) {
	res, err := getNameplates(t, &stubQuery{answer: &playerv1.GetPlayersResponse{}}, adaID)

	require.NoError(t, err)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"), "the page keeps what it read, and learns its own rename")
}

func TestAFailedReadIsTheErrorNets(t *testing.T) {
	failure := errors.New("postgres is down")

	_, err := getNameplates(t, &stubQuery{err: failure}, adaID)

	assert.ErrorIs(t, err, failure)
}
