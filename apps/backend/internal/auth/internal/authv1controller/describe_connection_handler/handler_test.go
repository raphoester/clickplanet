package describe_connection_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/authv1controller/describe_connection_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/connections/usecases/describe_connection_usecase"
)

type stub struct{}

func (stub) Execute(_ context.Context, sealed string) (describe_connection_usecase.Out, error) {
	switch sealed {
	case "lapsed":
		return describe_connection_usecase.Out{}, fmt.Errorf("%w: ten minutes ago", connections.ErrExpired)
	case "made up":
		return describe_connection_usecase.Out{}, fmt.Errorf("%w: not sealed here", connections.ErrInvalidRequest)
	case "broken":
		return describe_connection_usecase.Out{}, errors.New("on fire")
	}
	return describe_connection_usecase.Out{Client: "Claude", RedirectHost: "claude.ai", DeniedURL: "https://claude.ai/cb?error=access_denied"}, nil
}

func describe(t *testing.T, request string) (*connect.Response[authv1.DescribeConnectionResponse], error) {
	t.Helper()
	return describe_connection_handler.New(stub{}).DescribeConnection(t.Context(), //nolint:wrapcheck // the tests read the code.
		connect.NewRequest(&authv1.DescribeConnectionRequest{Request: request}))
}

func TestTheOperatorIsShownWhoAsksAndWhereItIs(t *testing.T) {
	res, err := describe(t, "sealed")

	require.NoError(t, err)
	assert.Equal(t, "Claude", res.Msg.GetClientName())
	assert.Equal(t, "claude.ai", res.Msg.GetRedirectHost())
	assert.Equal(t, "https://claude.ai/cb?error=access_denied", res.Msg.GetDenyUrl())
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func TestARequestToStartAgainIsTellableFromOneToForget(t *testing.T) {
	_, err := describe(t, "lapsed")
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

	_, err = describe(t, "made up")
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = describe(t, "broken")
	require.Error(t, err)
	var coded *connect.Error
	assert.NotErrorAs(t, err, &coded, "a failure is the error net's to answer")
}
