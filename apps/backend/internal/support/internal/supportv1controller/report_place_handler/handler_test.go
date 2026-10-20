package report_place_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places/usecases/report_place_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/report_place_handler"
)

var ada = places.AccountID{15: 1}

type stubUseCase struct {
	err   error
	asked []report_place_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in report_place_usecase.In) error {
	s.asked = append(s.asked, in)
	return s.err
}

func report(ctx context.Context, useCase *stubUseCase, msg *supportv1.ReportPlaceRequest) error {
	if _, err := report_place_handler.New(useCase).ReportPlace(ctx, connect.NewRequest(msg)); err != nil {
		return fmt.Errorf("ReportPlace failed: %w", err)
	}
	return nil
}

func asAda(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), ada.String())
}

func TestTheReportReachesTheUseCaseUnderTheCallersAccount(t *testing.T) {
	useCase := &stubUseCase{}

	err := report(asAda(t), useCase, &supportv1.ReportPlaceRequest{
		TileId: 42, Reason: supportv1.PlaceReportReason_PLACE_REPORT_REASON_HATE, Text: "a swastika",
	})
	require.NoError(t, err)

	assert.Equal(t, []report_place_usecase.In{{
		Reporter: ada, Tile: 42, Complaint: places.Complaint{Reason: places.ReasonHate, Text: "a swastika"},
	}}, useCase.asked)
}

func TestEveryReasonTheProtoNamesIsAReason(t *testing.T) {
	for value, name := range supportv1.PlaceReportReason_name {
		reason := supportv1.PlaceReportReason(value)
		if reason == supportv1.PlaceReportReason_PLACE_REPORT_REASON_UNSPECIFIED {
			continue
		}
		useCase := &stubUseCase{}

		require.NoError(t, report(asAda(t), useCase, &supportv1.ReportPlaceRequest{TileId: 42, Reason: reason}), name)

		_, err := places.NewReport(places.ReportID{15: 1}, ada, 42, useCase.asked[0].Complaint, time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC))
		require.NoError(t, err, name)
	}
}

func TestNoAccountIsUnauthenticatedAndNothingIsAsked(t *testing.T) {
	useCase := &stubUseCase{}

	err := report(t.Context(), useCase, &supportv1.ReportPlaceRequest{TileId: 42})

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, useCase.asked)
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		fmt.Errorf("refused: %w", places.ErrUnknownReason): connect.CodeInvalidArgument,
		fmt.Errorf("refused: %w", notes.ErrInvalidText):    connect.CodeInvalidArgument,
		fmt.Errorf("refused: %w", places.ErrInvalidReport): connect.CodeInvalidArgument,
		assert.AnError: connect.CodeUnknown,
	} {
		got := report(asAda(t), &stubUseCase{err: err}, &supportv1.ReportPlaceRequest{TileId: 42})
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
