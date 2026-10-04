package wear_title_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/wear_title_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubUseCase struct {
	worn  titles.Standing
	err   error
	asked titles.ID
}

func (s *stubUseCase) Execute(_ context.Context, _ players.AccountID, title titles.ID) (titles.Standing, error) {
	s.asked = title
	return s.worn, s.err
}

func wear(t *testing.T, ctx context.Context, useCase *stubUseCase, id string) (*connect.Response[playerv1.WearTitleResponse], error) {
	t.Helper()

	return wear_title_handler.New(useCase).WearTitle(ctx, connect.NewRequest(&playerv1.WearTitleRequest{TitleId: id})) //nolint:wrapcheck // the tests read the connect error.
}

func signedIn(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), players.AccountID{15: 1}.String())
}

func TestTheTitleWornIsAnswered(t *testing.T) {
	og, _ := titles.NewCatalog().StandingOf("og")
	useCase := &stubUseCase{worn: og}

	res, err := wear(t, signedIn(t), useCase, "og")

	require.NoError(t, err)
	assert.Equal(t, titles.ID("og"), useCase.asked)
	assert.Equal(t, "og", res.Msg.GetWorn().GetId())
}

func TestATitleThatCannotBeWornIsInvalid(t *testing.T) {
	_, err := wear(t, signedIn(t), &stubUseCase{err: fmt.Errorf("wrapped: %w", wearing.ErrNotWearable)}, "emperor")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestNoAccountIsUnauthenticatedAndAFailureIsNotTheCallers(t *testing.T) {
	_, err := wear(t, t.Context(), &stubUseCase{}, "og")
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = wear(t, signedIn(t), &stubUseCase{err: errors.New("postgres is down")}, "og")
	require.Error(t, err)
	assert.NotEqual(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
