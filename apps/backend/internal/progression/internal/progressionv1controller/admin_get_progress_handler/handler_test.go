package admin_get_progress_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/admin_get_progress_handler"
)

type stubQuery struct {
	asked []accounts.AccountID
}

func (s *stubQuery) Progress(_ context.Context, account accounts.AccountID) (*progressionv1.AdminGetProgressResponse, error) {
	s.asked = append(s.asked, account)
	return &progressionv1.AdminGetProgressResponse{}, nil
}

func TestTheOperatorReadsAnAccountsProgressUnstored(t *testing.T) {
	query := &stubQuery{}

	res, err := admin_get_progress_handler.New(query).GetProgress(t.Context(),
		connect.NewRequest(&progressionv1.AdminGetProgressRequest{AccountId: accounts.AccountID{15: 1}.String()}))

	require.NoError(t, err)
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
	assert.Equal(t, []accounts.AccountID{{15: 1}}, query.asked)
}

func TestAnIdThatIsNotAnAccountIsRefused(t *testing.T) {
	query := &stubQuery{}

	_, err := admin_get_progress_handler.New(query).GetProgress(t.Context(),
		connect.NewRequest(&progressionv1.AdminGetProgressRequest{AccountId: "nope"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, query.asked)
}
