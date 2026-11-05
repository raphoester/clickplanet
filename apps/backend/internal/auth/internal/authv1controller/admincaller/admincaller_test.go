package admincaller_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/admincaller"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/get_operator_handler/operator_query"
)

type operators map[string]error

func (o operators) Operator(_ context.Context, cookieHeader string) (*authv1.GetOperatorResponse, error) {
	if err, refused := o[cookieHeader]; refused {
		return nil, err
	}
	return &authv1.GetOperatorResponse{AccountId: accounts.AccountID{15: 1}.String()}, nil
}

func header(cookie string) http.Header {
	header := http.Header{}
	header.Set("Cookie", cookie)
	return header
}

func TestTheCookieNamesTheOperator(t *testing.T) {
	account, err := admincaller.AccountOf(t.Context(), operators{}, header("cp_admin=ada"))

	require.NoError(t, err)
	assert.Equal(t, accounts.AccountID{15: 1}, account)
}

func TestARefusalIsAnsweredWithTheCodeOfItsReason(t *testing.T) {
	refusals := operators{
		"":             operator_query.ErrNoSession,
		"cp_admin=bob": operator_query.ErrNotAnOperator,
		"cp_admin=x":   errors.New("postgres is down"),
	}

	for cookie, code := range map[string]connect.Code{
		"":             connect.CodeUnauthenticated,
		"cp_admin=bob": connect.CodePermissionDenied,
		"cp_admin=x":   connect.CodeUnknown,
	} {
		_, err := admincaller.AccountOf(t.Context(), refusals, header(cookie))
		assert.Equal(t, code, connect.CodeOf(err), cookie)
	}
}
