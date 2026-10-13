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
	return send(ctx, useCase, &supportv1.SuggestFeatureRequest{Text: text}, "")
}

func send(ctx context.Context, useCase *stubUseCase, msg *supportv1.SuggestFeatureRequest, userAgent string) error {
	req := connect.NewRequest(msg)
	req.Header().Set("User-Agent", userAgent)
	if _, err := suggest_feature_handler.New(useCase).SuggestFeature(ctx, req); err != nil {
		return fmt.Errorf("SuggestFeature failed: %w", err)
	}
	return nil
}

func asAda(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), ada.String())
}

func TestTheCallersAccountTheTextThePageAndTheBrowserAreHandedOn(t *testing.T) {
	useCase := &stubUseCase{}

	err := send(asAda(t), useCase, &supportv1.SuggestFeatureRequest{
		Text:    "a night mode",
		Context: &supportv1.PageContext{Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, GpuRenderer: "Apple GPU"},
	}, "Mozilla/5.0 (iPhone)")
	require.NoError(t, err)

	assert.Equal(t, []suggest_feature_usecase.In{{
		Account: ada,
		Text:    "a night mode",
		Diagnostics: notes.Diagnostics{
			Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, GPURenderer: "Apple GPU", UserAgent: "Mozilla/5.0 (iPhone)",
		},
	}}, useCase.asked)
}

func TestASuggestionWithNoContextKeepsTheBrowserAlone(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, send(asAda(t), useCase, &supportv1.SuggestFeatureRequest{Text: "a night mode"}, "Mozilla/5.0"))

	assert.Equal(t, notes.Diagnostics{UserAgent: "Mozilla/5.0"}, useCase.asked[0].Diagnostics)
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
