package report_player_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/usecases/report_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/report_player_handler"
)

var (
	ada = reports.AccountID{15: 1}
	bob = reports.AccountID{15: 2}
)

type stubUseCase struct {
	err   error
	asked []report_player_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in report_player_usecase.In) error {
	s.asked = append(s.asked, in)
	return s.err
}

func report(ctx context.Context, useCase *stubUseCase, msg *supportv1.ReportPlayerRequest) error {
	if _, err := report_player_handler.New(useCase).ReportPlayer(ctx, connect.NewRequest(msg)); err != nil {
		return fmt.Errorf("ReportPlayer failed: %w", err)
	}
	return nil
}

func asAda(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), ada.String())
}

func TestTheReportReachesTheUseCaseUnderTheCallersAccount(t *testing.T) {
	useCase := &stubUseCase{}

	err := report(asAda(t), useCase, &supportv1.ReportPlayerRequest{
		AccountId: bob.String(), Reason: supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE, Text: "it insults everyone", MessageId: "message-1",
	})
	require.NoError(t, err)

	assert.Equal(t, []report_player_usecase.In{{
		Reporter: ada, Reported: bob,
		Complaint: reports.Complaint{Reason: reports.ReasonChatAbuse, Text: "it insults everyone", Message: "message-1"},
	}}, useCase.asked)
}

func reportOf(complaint reports.Complaint) error {
	_, err := reports.NewReport(reports.ReportID{15: 1}, ada, bob, complaint, time.Now(), nil)
	return err //nolint:wrapcheck // the tests read the domain's own error.
}

func TestEveryReasonTheProtoNamesIsOneTheDomainKnows(t *testing.T) {
	for number, name := range supportv1.ReportReason_name {
		if supportv1.ReportReason(number) == supportv1.ReportReason_REPORT_REASON_UNSPECIFIED {
			continue
		}
		useCase := &stubUseCase{}

		require.NoError(t, report(asAda(t), useCase, &supportv1.ReportPlayerRequest{AccountId: bob.String(), Reason: supportv1.ReportReason(number)}))

		assert.NoError(t, reportOf(useCase.asked[0].Complaint), name)
	}
}

func TestNoReasonAndAReasonTheProtoDoesNotNameAreOnesTheDomainRefuses(t *testing.T) {
	for _, reason := range []supportv1.ReportReason{supportv1.ReportReason_REPORT_REASON_UNSPECIFIED, 99} {
		useCase := &stubUseCase{}

		require.NoError(t, report(asAda(t), useCase, &supportv1.ReportPlayerRequest{AccountId: bob.String(), Reason: reason}))

		assert.ErrorIs(t, reportOf(useCase.asked[0].Complaint), reports.ErrUnknownReason, reason.String())
	}
}

func TestACallerWithNoAccountIsRefusedBeforeTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	err := report(t.Context(), useCase, &supportv1.ReportPlayerRequest{AccountId: bob.String()})

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, useCase.asked)
}

func TestAnIDThatIsNotAnAccountIsRefusedBeforeTheUseCase(t *testing.T) {
	for _, id := range []string{"", "guest_a1b2c3", "00000000-0000-0000-0000-000000000000"} {
		useCase := &stubUseCase{}

		err := report(asAda(t), useCase, &supportv1.ReportPlayerRequest{AccountId: id})

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), id)
		assert.Empty(t, useCase.asked, id)
	}
}

func TestTheRefusalsHaveTheirCodes(t *testing.T) {
	for refusal, want := range map[error]connect.Code{
		reports.ErrUnknownReason:   connect.CodeInvalidArgument,
		notes.ErrInvalidText:       connect.CodeInvalidArgument,
		reports.ErrOwnAccount:      connect.CodeInvalidArgument,
		reports.ErrReportedAlready: connect.CodeAlreadyExists,
	} {
		err := report(asAda(t), &stubUseCase{err: fmt.Errorf("wrapped: %w", refusal)},
			&supportv1.ReportPlayerRequest{AccountId: bob.String()})

		assert.Equal(t, want, connect.CodeOf(err), "%v", refusal)
	}
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	err := report(asAda(t), &stubUseCase{err: errors.New("postgres is down")}, &supportv1.ReportPlayerRequest{AccountId: bob.String()})

	assert.ErrorContains(t, err, "postgres is down")
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
