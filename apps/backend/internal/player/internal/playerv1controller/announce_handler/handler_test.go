package announce_handler_test

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/announce_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence/usecases/announce_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct {
	in  *announce_usecase.In
	err error
}

func (s stubUseCase) Execute(_ context.Context, in announce_usecase.In) error {
	*s.in = in
	return s.err
}

func announce(ctx context.Context, useCase stubUseCase) error {
	_, err := announce_handler.New(useCase).Announce(ctx,
		connect.NewRequest(&playerv1.AnnounceRequest{CountryId: "fr"}))
	return err //nolint:wrapcheck // the test reads the handler's own error.
}

func callerContext(t *testing.T) context.Context {
	t.Helper()
	return cpctx.AddIPToContext(cpctx.AddAccountToContext(t.Context(), "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"), "1.2.3.4")
}

func TestTheRequestTheCallerAndItsAddressAreMapped(t *testing.T) {
	var in announce_usecase.In

	require.NoError(t, announce(callerContext(t), stubUseCase{in: &in}))

	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", in.Account.String())
	assert.Equal(t, "fr", in.Country)
	assert.Equal(t, "1.2.3.4", in.IP)
}

func TestACallerWithNoAccountIsUnauthenticated(t *testing.T) {
	err := announce(t.Context(), stubUseCase{in: &announce_usecase.In{}})

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestAnUnknownCountryIsInvalidArgumentAndSaysNoMore(t *testing.T) {
	err := announce(callerContext(t), stubUseCase{
		in:  &announce_usecase.In{},
		err: fmt.Errorf("%w: %q", presence.ErrUnknownCountry, "atlantis"),
	})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), "atlantis")
}

func TestAnUnexpectedFailureIsLeftForTheErrorNet(t *testing.T) {
	err := announce(callerContext(t), stubUseCase{in: &announce_usecase.In{}, err: assert.AnError})

	require.ErrorIs(t, err, assert.AnError)
}
