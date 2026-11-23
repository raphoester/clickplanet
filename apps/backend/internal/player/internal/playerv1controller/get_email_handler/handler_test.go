package get_email_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_email_handler"
)

type stubQuery struct {
	asked []players.AccountID
}

func (s *stubQuery) Email(_ context.Context, account players.AccountID) (*playerv1.GetEmailResponse, error) {
	s.asked = append(s.asked, account)
	email := "ada@example.com"
	return &playerv1.GetEmailResponse{Email: &email}, nil
}

func TestTheAddressIsWhatTheQueryReadForTheAccount(t *testing.T) {
	query := &stubQuery{}

	res, err := get_email_handler.New(query).GetEmail(t.Context(),
		connect.NewRequest(&playerv1.GetEmailRequest{AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"}))

	require.NoError(t, err)
	assert.Equal(t, "ada@example.com", res.Msg.GetEmail())
	assert.Len(t, query.asked, 1)
}

func TestAnIDThatIsNotAnAccountIsRefusedAndAuthIsNotAsked(t *testing.T) {
	query := &stubQuery{}

	_, err := get_email_handler.New(query).GetEmail(t.Context(), connect.NewRequest(&playerv1.GetEmailRequest{AccountId: "nobody"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, query.asked)
}
