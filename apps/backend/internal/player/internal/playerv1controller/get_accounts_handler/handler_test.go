package get_accounts_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_accounts_handler"
)

var (
	ada   = players.AccountID{15: 1}
	guest = players.AccountID{15: 2}
	ghost = players.AccountID{15: 3}
	made  = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
)

type stubAccounts struct {
	known map[players.AccountID]players.Account
	err   error
	asked [][]players.AccountID
}

func (s *stubAccounts) Accounts(_ context.Context, accounts []players.AccountID) (map[players.AccountID]players.Account, error) {
	s.asked = append(s.asked, accounts)
	found := map[players.AccountID]players.Account{}
	for _, id := range accounts {
		if account, ok := s.known[id]; ok {
			found[id] = account
		}
	}
	return found, s.err
}

func getAccounts(t *testing.T, accounts *stubAccounts, ids ...string) (*connect.Response[playerv1.GetAccountsResponse], error) {
	t.Helper()

	return get_accounts_handler.New(accounts).GetAccounts(t.Context(), connect.NewRequest(&playerv1.GetAccountsRequest{AccountIds: ids})) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestEachAccountAuthKnowsIsAnsweredOnceInTheOrderAsked(t *testing.T) {
	accounts := &stubAccounts{known: map[players.AccountID]players.Account{
		ada:   players.AccountOf(true, made),
		guest: players.AccountOf(false, time.Time{}),
	}}

	res, err := getAccounts(t, accounts, guest.String(), ghost.String(), ada.String(), guest.String())

	require.NoError(t, err)
	want := &playerv1.GetAccountsResponse{Accounts: []*playerv1.Account{
		{AccountId: guest.String()},
		{AccountId: ada.String(), Linked: true, CreatedAtUnixMs: made.UnixMilli()},
	}}
	assert.True(t, proto.Equal(want, res.Msg), "got %v", res.Msg)
}

func TestAnIdThatIsNotAnAccountIsRefusedAndReadsNothing(t *testing.T) {
	for _, id := range []string{"", "not-an-account", "00000000-0000-0000-0000-000000000000"} {
		accounts := &stubAccounts{}

		_, err := getAccounts(t, accounts, ada.String(), id)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
		assert.Empty(t, accounts.asked, id)
	}
}
