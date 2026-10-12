package suggest_feature_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/suggest_feature_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/suggest_feature_handler"
)

var ada = notes.AccountID{15: 1}

type stubUseCase struct {
	err   error
	asked []suggest_feature_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in suggest_feature_usecase.In) error {
	s.asked = append(s.asked, in)
	return s.err
}

func suggest(ctx context.Context, useCase *stubUseCase, text string) error {
	_, err := suggest_feature_handler.New(useCase).SuggestFeature(ctx,
		connect.NewRequest(&supportv1.SuggestFeatureRequest{Text: text}))
	if err != nil {
		return fmt.Errorf("SuggestFeature failed: %w", err)
	}
	return nil
}

func asAda(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), ada.String())
}

func TestTheCallersAccountAndTheTextAreHandedOn(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, suggest(asAda(t), useCase, "a night mode"))

	assert.Equal(t, []suggest_feature_usecase.In{{Account: ada, Text: "a night mode"}}, useCase.asked)
}

func TestACallerWithNoAccountIsRefusedBeforeTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	err := suggest(t.Context(), useCase, "a night mode")

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, useCase.asked)
}

func TestAnInvalidTextIsTheCallersFault(t *testing.T) {
	err := suggest(asAda(t), &stubUseCase{err: fmt.Errorf("wrapped: %w", notes.ErrInvalidText)}, " ")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAStoreFailureIsLeftToTheErrorNet(t *testing.T) {
	err := suggest(asAda(t), &stubUseCase{err: errors.New("postgres is down")}, "a night mode")

	assert.ErrorContains(t, err, "postgres is down")
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
