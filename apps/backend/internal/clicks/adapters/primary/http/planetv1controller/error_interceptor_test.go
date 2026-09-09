package planetv1controller

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/stretchr/testify/require"
)

func intercept(handlerErr error) error {
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		if handlerErr != nil {
			return nil, handlerErr
		}
		return connect.NewResponse(&planetv1.ClickResponse{}), nil
	})

	_, err := NewErrorInterceptor(nil).WrapUnary(next)(
		context.Background(), connect.NewRequest(&planetv1.ClickRequest{}))

	return err
}

func TestErrorInterceptor(t *testing.T) {
	t.Run("lets a success through", func(t *testing.T) {
		require.NoError(t, intercept(nil))
	})

	t.Run("keeps a code the handler picked itself", func(t *testing.T) {
		chosen := connect.NewError(connect.CodeNotFound, errors.New("no such tile"))
		err := intercept(chosen)
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
		require.Contains(t, err.Error(), "no such tile")
	})
}
