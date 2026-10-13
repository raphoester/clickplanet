package report_bug_handler_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/report_bug_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/report_bug_handler"
)

var ada = notes.AccountID{15: 1}

type stubUseCase struct {
	out   report_bug_usecase.Out
	err   error
	asked []report_bug_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in report_bug_usecase.In) (report_bug_usecase.Out, error) {
	s.asked = append(s.asked, in)
	return s.out, s.err
}

func report(ctx context.Context, useCase *stubUseCase, msg *supportv1.ReportBugRequest, userAgent string) error {
	_, err := answered(ctx, useCase, msg, userAgent)
	return err
}

func answered(
	ctx context.Context, useCase *stubUseCase, msg *supportv1.ReportBugRequest, userAgent string,
) (*supportv1.ReportBugResponse, error) {
	req := connect.NewRequest(msg)
	req.Header().Set("User-Agent", userAgent)
	res, err := report_bug_handler.New(useCase).ReportBug(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ReportBug failed: %w", err)
	}
	return res.Msg, nil
}

func asAda(t *testing.T) context.Context {
	t.Helper()

	return cpctx.AddAccountToContext(t.Context(), ada.String())
}

func TestTheTextThePageAndTheBrowserAreHandedOn(t *testing.T) {
	useCase := &stubUseCase{}

	err := report(asAda(t), useCase, &supportv1.ReportBugRequest{
		Text:    "the globe is black",
		Context: &supportv1.PageContext{Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, ConnectionDown: true},
	}, "Mozilla/5.0 (iPhone)")
	require.NoError(t, err)

	assert.Equal(t, []report_bug_usecase.In{{
		Account: ada,
		Text:    "the globe is black",
		Diagnostics: notes.Diagnostics{
			Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, ConnectionDown: true, UserAgent: "Mozilla/5.0 (iPhone)",
		},
	}}, useCase.asked)
}

func TestTheFilesAreHandedOnAndEachUploadLinkIsAnsweredInItsOrder(t *testing.T) {
	useCase := &stubUseCase{out: report_bug_usecase.Out{UploadURLs: []string{"https://bucket.example/a/0", "https://bucket.example/a/1"}}}

	res, err := answered(asAda(t), useCase, &supportv1.ReportBugRequest{
		Text: "the globe is black",
		Attachments: []*supportv1.Attachment{
			{ContentType: "image/png", SizeBytes: 400}, {ContentType: "video/mp4", SizeBytes: 9000},
		},
	}, "")
	require.NoError(t, err)

	assert.Equal(t, []notes.Attachment{
		{MediaType: notes.MediaPNG, Size: 400}, {MediaType: notes.MediaMP4, Size: 9000},
	}, useCase.asked[0].Attachments)
	require.Len(t, res.GetUploads(), 2)
	assert.Equal(t, "https://bucket.example/a/0", res.GetUploads()[0].GetUrl())
	assert.Equal(t, "https://bucket.example/a/1", res.GetUploads()[1].GetUrl())
}

func TestAReportWithNoFileAnswersNoUpload(t *testing.T) {
	res, err := answered(asAda(t), &stubUseCase{}, &supportv1.ReportBugRequest{Text: "the globe is black"}, "")

	require.NoError(t, err)
	assert.Empty(t, res.GetUploads())
}

func TestFilesTheRulesDoNotTakeAreTheCallersFault(t *testing.T) {
	err := report(asAda(t), &stubUseCase{err: fmt.Errorf("wrapped: %w", notes.ErrInvalidAttachment)},
		&supportv1.ReportBugRequest{Text: "the globe is black"}, "")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAReportWithNoContextKeepsTheBrowserAlone(t *testing.T) {
	useCase := &stubUseCase{}

	require.NoError(t, report(asAda(t), useCase, &supportv1.ReportBugRequest{Text: "nothing happens"}, "Mozilla/5.0"))

	assert.Equal(t, notes.Diagnostics{UserAgent: "Mozilla/5.0"}, useCase.asked[0].Diagnostics)
}

func TestACallerWithNoAccountIsRefusedBeforeTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	err := report(t.Context(), useCase, &supportv1.ReportBugRequest{Text: "the globe is black"}, "")

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, useCase.asked)
}

func TestAnInvalidTextIsTheCallersFault(t *testing.T) {
	err := report(asAda(t), &stubUseCase{err: fmt.Errorf("wrapped: %w", notes.ErrInvalidText)},
		&supportv1.ReportBugRequest{Text: " "}, "")

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAStoreFailureIsLeftToTheErrorNet(t *testing.T) {
	err := report(asAda(t), &stubUseCase{err: errors.New("postgres is down")},
		&supportv1.ReportBugRequest{Text: "the globe is black"}, "")

	assert.ErrorContains(t, err, "postgres is down")
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
