package cpconnect_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

const procedure = "/test.v1.TestService/Do"

func intercept(mapper cpconnect.Mapper, handlerErr error) error {
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		if handlerErr != nil {
			return nil, handlerErr
		}

		return connect.NewResponse(&emptypb.Empty{}), nil
	})

	_, err := cpconnect.NewErrorInterceptor(nil, mapper).WrapUnary(next)(
		context.Background(), connect.NewRequest(&emptypb.Empty{}))

	return err
}

func TestErrorInterceptor(t *testing.T) {
	t.Run("lets a success through", func(t *testing.T) {
		require.NoError(t, intercept(nil, nil))
	})

	t.Run("keeps a code the handler picked itself", func(t *testing.T) {
		chosen := connect.NewError(connect.CodeNotFound, errors.New("no such tile"))

		err := intercept(nil, chosen)

		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
		require.Contains(t, err.Error(), "no such tile")
	})

	// Handlers map the sentinels they recognise themselves; anything reaching
	// here is by definition unexpected, and its cause is the server's business.
	t.Run("an unrecognised error is internal and does not leak the cause", func(t *testing.T) {
		err := intercept(nil, errors.New("disk on fire"))

		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		require.NotContains(t, err.Error(), "disk on fire")
	})

	t.Run("a mapper that recognises the error picks the code", func(t *testing.T) {
		sentinel := errors.New("not a real message")
		mapper := func(err error) *connect.Error {
			if errors.Is(err, sentinel) {
				return connect.NewError(connect.CodeInvalidArgument, err)
			}

			return nil
		}

		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(intercept(mapper, sentinel)))
		require.Equal(t, connect.CodeInternal, connect.CodeOf(intercept(mapper, errors.New("other"))))
	})
}

type fakeStreamConn struct {
	connect.StreamingHandlerConn
}

func (fakeStreamConn) Spec() connect.Spec {
	return connect.Spec{Procedure: procedure}
}

// This is why NewErrorInterceptor is a full connect.Interceptor and not a
// UnaryInterceptorFunc: without WrapStreamingHandler, a live feed would be the
// one procedure whose raw error reached the caller.
func TestErrorInterceptorCoversStreams(t *testing.T) {
	next := connect.StreamingHandlerFunc(func(context.Context, connect.StreamingHandlerConn) error {
		return errors.New("disk on fire")
	})

	err := cpconnect.NewErrorInterceptor(nil, nil).WrapStreamingHandler(next)(t.Context(), fakeStreamConn{})

	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "disk on fire")
}
