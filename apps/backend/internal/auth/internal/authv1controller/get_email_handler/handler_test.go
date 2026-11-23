package get_email_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_email_handler"
)

type stubQuery struct {
	asked []accounts.AccountID
}

func (s *stubQuery) Email(_ context.Context, account accounts.AccountID) (*authv1.GetEmailResponse, error) {
	s.asked = append(s.asked, account)
	email := "ada@example.com"
	return &authv1.GetEmailResponse{Email: &email}, nil
}

func TestTheAddressIsWhatTheQueryReadForTheAccount(t *testing.T) {
	query := &stubQuery{}

	res, err := get_email_handler.New(query).GetEmail(t.Context(),
		connect.NewRequest(&authv1.GetEmailRequest{AccountId: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"}))

	require.NoError(t, err)
	assert.Equal(t, "ada@example.com", res.Msg.GetEmail())
	want, err := accounts.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	assert.Equal(t, []accounts.AccountID{want}, query.asked)
}

func TestAnIDThatIsNotAnAccountIsRefusedAndNobodyIsAsked(t *testing.T) {
	query := &stubQuery{}

	_, err := get_email_handler.New(query).GetEmail(t.Context(), connect.NewRequest(&authv1.GetEmailRequest{AccountId: "not-a-uuid"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, query.asked)
}
