package e2e_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1/supportv1connect"
)

func (p *gamer) support() supportv1connect.SupportServiceClient {
	return supportv1connect.NewSupportServiceClient(http.DefaultClient, p.stack.baseURL)
}

func (p *gamer) suggest(text string) error {
	p.t.Helper()

	return p.suggestFrom(text, nil, "")
}

func (p *gamer) suggestFrom(text string, page *supportv1.PageContext, userAgent string) error {
	p.t.Helper()

	req := connect.NewRequest(&supportv1.SuggestFeatureRequest{Text: text, Context: page})
	p.send(req.Header())
	req.Header().Set("User-Agent", userAgent)
	if _, err := p.support().SuggestFeature(p.t.Context(), req); err != nil {
		return fmt.Errorf("SuggestFeature failed: %w", err)
	}
	return nil
}

func (p *gamer) reportBug(text string, page *supportv1.PageContext, userAgent string) error {
	p.t.Helper()

	req := connect.NewRequest(&supportv1.ReportBugRequest{Text: text, Context: page})
	p.send(req.Header())
	req.Header().Set("User-Agent", userAgent)
	if _, err := p.support().ReportBug(p.t.Context(), req); err != nil {
		return fmt.Errorf("ReportBug failed: %w", err)
	}
	return nil
}

type file struct {
	contentType string
	body        []byte
}

func (p *gamer) createUpload(sent file) (*supportv1.CreateUploadResponse, error) {
	p.t.Helper()

	req := connect.NewRequest(&supportv1.CreateUploadRequest{ContentType: sent.contentType, SizeBytes: uint64(len(sent.body))})
	p.send(req.Header())
	res, err := p.support().CreateUpload(p.t.Context(), req)
	if err != nil {
		return nil, fmt.Errorf("CreateUpload failed: %w", err)
	}
	return res.Msg, nil
}

func (p *gamer) uploaded(files ...file) []string {
	p.t.Helper()

	ids := make([]string, 0, len(files))
	for _, sent := range files {
		upload, err := p.createUpload(sent)
		require.NoError(p.t, err)
		put(p.t, upload.GetUrl(), sent)
		ids = append(ids, upload.GetAttachmentId())
	}
	return ids
}

func (p *gamer) reportBugWith(text string, ids ...string) error {
	p.t.Helper()

	req := connect.NewRequest(&supportv1.ReportBugRequest{Text: text, AttachmentIds: ids})
	p.send(req.Header())
	if _, err := p.support().ReportBug(p.t.Context(), req); err != nil {
		return fmt.Errorf("ReportBug failed: %w", err)
	}
	return nil
}

func put(t *testing.T, link string, sent file) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPut, link, bytes.NewReader(sent.body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", sent.contentType)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func opened(t *testing.T, attachment *supportv1.NoteAttachment) (int, file) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, attachment.GetUrl(), nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return res.StatusCode, file{contentType: res.Header.Get("Content-Type"), body: body}
}

var (
	screenshot = file{contentType: "image/png", body: []byte("a screenshot")}
	recording  = file{contentType: "video/mp4", body: []byte("a screen recording")}
)

func TestThePageIsToldWhatANoteCarriesWithNoSession(t *testing.T) {
	game := startGame(t)

	res, err := supportv1connect.NewSupportServiceClient(http.DefaultClient, game.baseURL).GetAttachmentRules(t.Context(),
		connect.NewRequest(&supportv1.GetAttachmentRulesRequest{}))

	require.NoError(t, err)
	assert.Equal(t, uint32(10), res.Msg.GetMaxPerNote())
	assert.Equal(t, uint64(50<<20), res.Msg.GetMaxBytes())
	assert.Contains(t, res.Msg.GetContentTypes(), "image/png")
	assert.Contains(t, res.Msg.GetContentTypes(), "video/quicktime")
}

func TestAGuestUploadsItsFilesThenReportsABugAndTheOperatorOpensThem(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)

	require.NoError(t, guest.reportBugWith("the globe is black", guest.uploaded(screenshot, recording)...))

	bugs := game.notes(t)
	require.Len(t, bugs, 1)
	require.Len(t, bugs[0].GetAttachments(), 2)
	assert.Equal(t, "image/png", bugs[0].GetAttachments()[0].GetContentType())
	assert.Equal(t, uint64(len(recording.body)), bugs[0].GetAttachments()[1].GetSizeBytes())
	for n, sent := range []file{screenshot, recording} {
		status, got := opened(t, bugs[0].GetAttachments()[n])
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, sent, got)
	}
}

func TestASuggestionCarriesFilesAsABugReportDoes(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)

	req := connect.NewRequest(&supportv1.SuggestFeatureRequest{Text: "a night mode", AttachmentIds: guest.uploaded(screenshot)})
	guest.send(req.Header())
	_, err := guest.support().SuggestFeature(t.Context(), req)
	require.NoError(t, err)

	status, got := opened(t, game.notes(t)[0].GetAttachments()[0])
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, screenshot, got)
}

func TestANoteNamingAFileThatWasNeverUploadedIsRefusedAndNotKept(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)
	sent := guest.uploaded(screenshot)
	never, err := guest.createUpload(recording)
	require.NoError(t, err)

	err = guest.reportBugWith("the globe is black", sent[0], never.GetAttachmentId())

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.Empty(t, game.notes(t))
}

func TestAFileTheRulesDoNotTakeGetsNoUploadLink(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)

	_, err := guest.createUpload(file{contentType: "text/html", body: []byte("<script>")})
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "not a photo or a video")

	_, err = guest.createUpload(file{contentType: "image/png"})
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "an empty file")

	anonymous := connect.NewRequest(&supportv1.CreateUploadRequest{ContentType: "image/png", SizeBytes: 10})
	anonymous.Header().Set("X-Real-IP", callerIP)
	_, err = guest.support().CreateUpload(t.Context(), anonymous)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "no session")
}

func TestOneFileTooManyIsRefusedAndNoNoteIsKept(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)
	eleven := make([]file, 11)
	for n := range eleven {
		eleven[n] = screenshot
	}

	err := guest.reportBugWith("the globe is black", guest.uploaded(eleven...)...)

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, game.notes(t))
}

func TestADeletedAccountsNotesKeepTheirFilesAndNoLongerNameIt(t *testing.T) {
	game := startGame(t)
	leaving := game.newPlayer(t)
	require.NoError(t, leaving.reportBugWith("the globe is black", leaving.uploaded(screenshot, recording)...))

	deletion := connect.NewRequest(&authv1.DeleteAccountRequest{})
	leaving.send(deletion.Header())
	_, err := authv1connect.NewAuthServiceClient(http.DefaultClient, game.baseURL).DeleteAccount(t.Context(), deletion)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return game.notes(t)[0].GetAccountId() == ""
	}, 5*time.Second, 20*time.Millisecond)
	kept := game.notes(t)[0].GetAttachments()
	require.Len(t, kept, 2)
	for n, sent := range []file{screenshot, recording} {
		status, got := opened(t, kept[n])
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, sent, got)
	}
}

func (s gameStack) notes(t *testing.T) []*supportv1.Note {
	t.Helper()

	return s.notesOf(t, supportv1.NoteKind_NOTE_KIND_UNSPECIFIED)
}

func (s gameStack) notesOf(t *testing.T, kind supportv1.NoteKind) []*supportv1.Note {
	t.Helper()

	res, err := supportv1connect.NewAdminServiceClient(http.DefaultClient, s.consoleURL).ListNotes(t.Context(),
		connect.NewRequest(&supportv1.ListNotesRequest{Kind: kind}))
	require.NoError(t, err)
	return res.Msg.GetNotes()
}

func TestAGuestSuggestsAFeatureAndTheOperatorReadsItUnderItsAccount(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)

	require.NoError(t, guest.suggest("  a night mode\r\nfor the globe  "))

	notes := game.notes(t)
	require.Len(t, notes, 1)
	assert.Equal(t, supportv1.NoteKind_NOTE_KIND_SUGGESTION, notes[0].GetKind())
	assert.Equal(t, guest.account(), notes[0].GetAccountId())
	assert.Equal(t, "a night mode\nfor the globe", notes[0].GetText())
	assert.WithinDuration(t, time.Now(), notes[0].GetSubmittedAt().AsTime(), time.Minute)
}

func TestAGuestReportsABugAndTheOperatorReadsItWithThePageTheGraphicsAndTheBrowser(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)
	page := &supportv1.PageContext{
		Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, ConnectionDown: true, PixelRatio: 3,
		GpuVendor: "Apple Inc.", GpuRenderer: "Apple GPU", WebglVersion: "WebGL 2.0", MaxTextureSize: 4096,
		ContextLost: true, Cores: 6, MemoryGb: 4,
	}
	require.NoError(t, guest.suggestFrom("a night mode", page, "Mozilla/5.0 (iPhone)"))

	require.NoError(t, guest.reportBug("the globe is black", page, "Mozilla/5.0 (iPhone)"))

	bugs := game.notesOf(t, supportv1.NoteKind_NOTE_KIND_BUG)
	require.Len(t, bugs, 1)
	assert.Equal(t, supportv1.NoteKind_NOTE_KIND_BUG, bugs[0].GetKind())
	assert.Equal(t, guest.account(), bugs[0].GetAccountId())
	assert.Equal(t, "the globe is black", bugs[0].GetText())
	assert.True(t, proto.Equal(page, bugs[0].GetContext()), "want %v, got %v", page, bugs[0].GetContext())
	assert.Equal(t, "Mozilla/5.0 (iPhone)", bugs[0].GetUserAgent())

	every := game.notes(t)
	require.Len(t, every, 2)
	assert.True(t, proto.Equal(page, every[1].GetContext()), "a suggestion says the same of its page")
	assert.Equal(t, "Mozilla/5.0 (iPhone)", every[1].GetUserAgent())
}

func TestANoteNeedsAnAccountAndAText(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)

	suggestion := connect.NewRequest(&supportv1.SuggestFeatureRequest{Text: "a night mode"})
	suggestion.Header().Set("X-Real-IP", callerIP)
	_, err := guest.support().SuggestFeature(t.Context(), suggestion)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	bug := connect.NewRequest(&supportv1.ReportBugRequest{Text: "the globe is black"})
	bug.Header().Set("X-Real-IP", callerIP)
	_, err = guest.support().ReportBug(t.Context(), bug)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(guest.suggest("   ")))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(guest.reportBug("   ", nil, "")))
	assert.Empty(t, game.notes(t))
}

func TestAnAddressSendsThreeNotesAtOnceAndACallWithNoSessionSpendsNone(t *testing.T) {
	game := startGame(t)
	guest := game.newPlayer(t)

	for range 5 {
		anonymous := connect.NewRequest(&supportv1.SuggestFeatureRequest{Text: "a night mode"})
		anonymous.Header().Set("X-Real-IP", callerIP)
		_, err := guest.support().SuggestFeature(t.Context(), anonymous)
		require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	}

	ada := game.namedPlayer(t, "google-ada", "Ada")

	require.NoError(t, guest.suggest("an idea"))
	require.NoError(t, guest.reportBug("a bug", nil, ""))
	require.NoError(t, guest.reportPlayer(ada.account(), supportv1.ReportReason_REPORT_REASON_CHEATING, "", ""))
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(guest.suggest("one too many")))
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(guest.reportBug("one too many", nil, "")))
	assert.Equal(t, connect.CodeResourceExhausted,
		connect.CodeOf(guest.reportPlayer(ada.account(), supportv1.ReportReason_REPORT_REASON_CHEATING, "", "")),
		"suggestions, bug reports and player reports spend one allowance")
	assert.Len(t, game.notes(t), 2)
	assert.Len(t, game.reports(t), 1)
}

func TestADeletedAccountsNotesKeepTheirTextAndNoLongerNameIt(t *testing.T) {
	game := startGame(t)
	leaving, staying := game.newPlayer(t), game.newPlayer(t)
	require.NoError(t, leaving.suggest("a night mode"))
	require.NoError(t, staying.suggest("more flags"))

	deletion := connect.NewRequest(&authv1.DeleteAccountRequest{})
	leaving.send(deletion.Header())
	_, err := authv1connect.NewAuthServiceClient(http.DefaultClient, game.baseURL).DeleteAccount(t.Context(), deletion)
	require.NoError(t, err)

	senders := map[string]string{}
	require.Eventually(t, func() bool {
		for _, note := range game.notes(t) {
			senders[note.GetText()] = note.GetAccountId()
		}
		return senders["a night mode"] == ""
	}, 5*time.Second, 20*time.Millisecond)
	assert.Equal(t, map[string]string{"a night mode": "", "more flags": staying.account()}, senders)
}

func TestTheOperatorsListsAreNotOnThePublicRouter(t *testing.T) {
	game := startGame(t)
	public := supportv1connect.NewAdminServiceClient(http.DefaultClient, game.baseURL)

	_, err := public.ListNotes(t.Context(), connect.NewRequest(&supportv1.ListNotesRequest{}))
	assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))

	_, err = public.ListReports(t.Context(), connect.NewRequest(&supportv1.ListReportsRequest{}))
	assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func (p *gamer) reportPlayer(account string, reason supportv1.ReportReason, text string, messageID string) error {
	p.t.Helper()

	req := connect.NewRequest(&supportv1.ReportPlayerRequest{AccountId: account, Reason: reason, Text: text, MessageId: messageID})
	p.send(req.Header())
	if _, err := p.support().ReportPlayer(p.t.Context(), req); err != nil {
		return fmt.Errorf("ReportPlayer failed: %w", err)
	}
	return nil
}

func (s gameStack) reports(t *testing.T) []*supportv1.Report {
	t.Helper()

	res, err := supportv1connect.NewAdminServiceClient(http.DefaultClient, s.consoleURL).ListReports(t.Context(),
		connect.NewRequest(&supportv1.ListReportsRequest{}))
	require.NoError(t, err)
	return res.Msg.GetReports()
}

func TestAGuestReportsAPlayerByTheAccountItsMessageCarriesAndTheOperatorReadsWhoAndWhy(t *testing.T) {
	game := startGame(t)
	ada := game.namedPlayer(t, "google-ada", "Ada_L")
	said, err := ada.post()
	require.NoError(t, err)
	require.Equal(t, ada.account(), said.GetAuthorAccountId(), "the message says whose it is")
	guest := game.newPlayer(t)
	asGuest, err := guest.post()
	require.NoError(t, err)

	require.NoError(t, guest.reportPlayer(said.GetAuthorAccountId(), supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE,
		"  it insults everyone  ", said.GetId()))

	reports := game.reports(t)
	require.Len(t, reports, 1)
	assert.Equal(t, guest.account(), reports[0].GetReporterAccountId())
	assert.Equal(t, asGuest.GetAuthorName(), reports[0].GetReporterName())
	assert.Equal(t, ada.account(), reports[0].GetReportedAccountId())
	assert.Equal(t, "Ada_L", reports[0].GetReportedName())
	assert.Equal(t, supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE, reports[0].GetReason())
	assert.Equal(t, "it insults everyone", reports[0].GetText())
	assert.Equal(t, said.GetId(), reports[0].GetMessageId())
	assert.WithinDuration(t, time.Now(), reports[0].GetCreatedAt().AsTime(), time.Minute)
}

func TestAGuestIsReportedByTheAccountItsMessageCarries(t *testing.T) {
	game := startGame(t)
	spammer, reporter := game.newPlayer(t), game.newPlayer(t)
	said, err := spammer.post()
	require.NoError(t, err)
	require.Regexp(t, guestName, said.GetAuthorName())

	require.NoError(t, reporter.reportPlayer(said.GetAuthorAccountId(), supportv1.ReportReason_REPORT_REASON_OTHER, "", ""))

	reports := game.reports(t)
	require.Len(t, reports, 1)
	assert.Equal(t, spammer.account(), reports[0].GetReportedAccountId())
	assert.Equal(t, said.GetAuthorName(), reports[0].GetReportedName())
	assert.Empty(t, reports[0].GetText())
	assert.Empty(t, reports[0].GetMessageId())
}

func TestAReportFollowsThePlayerWhoRenamedAndNotWhoeverTookTheName(t *testing.T) {
	game := startGame(t)
	offender := game.namedPlayer(t, "google-ada", "Ada_L")
	said, err := offender.post()
	require.NoError(t, err)
	offender.choseItsNameAMonthAgo()
	_, err = offender.setName("Somebody_Else")
	require.NoError(t, err)
	bystander := game.namedPlayer(t, "google-bob", "Ada_L")
	reporter := game.newPlayer(t)

	require.NoError(t, reporter.reportPlayer(said.GetAuthorAccountId(), supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE, "", said.GetId()))

	reports := game.reports(t)
	require.Len(t, reports, 1)
	assert.Equal(t, offender.account(), reports[0].GetReportedAccountId())
	assert.NotEqual(t, bystander.account(), reports[0].GetReportedAccountId())
	assert.Equal(t, "Somebody_Else", reports[0].GetReportedName())
}

func TestAReportNeedsAnAccountAReasonAndAPlayerWhoIsNotTheReporter(t *testing.T) {
	game := startGame(t)
	ada := game.namedPlayer(t, "google-ada", "Ada_L")
	bob := game.namedPlayer(t, "google-bob", "Bob")

	anonymous := connect.NewRequest(&supportv1.ReportPlayerRequest{
		AccountId: bob.account(), Reason: supportv1.ReportReason_REPORT_REASON_CHEATING,
	})
	anonymous.Header().Set("X-Real-IP", callerIP)
	_, err := ada.support().ReportPlayer(t.Context(), anonymous)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	assert.Equal(t, connect.CodeInvalidArgument,
		connect.CodeOf(ada.reportPlayer(bob.account(), supportv1.ReportReason_REPORT_REASON_UNSPECIFIED, "", "")))
	assert.Equal(t, connect.CodeInvalidArgument,
		connect.CodeOf(ada.reportPlayer("Bob", supportv1.ReportReason_REPORT_REASON_CHEATING, "", "")), "a name is not an account")
	assert.Equal(t, connect.CodeInvalidArgument,
		connect.CodeOf(ada.reportPlayer(ada.account(), supportv1.ReportReason_REPORT_REASON_CHEATING, "", "")), "itself")
	assert.Empty(t, game.reports(t))
}

func TestAnAccountReportsAPlayerOnceADay(t *testing.T) {
	game := startGame(t)
	ada := game.namedPlayer(t, "google-ada", "Ada_L")
	bob := game.namedPlayer(t, "google-bob", "Bob")
	cyd := game.namedPlayer(t, "google-cyd", "Cyd")

	require.NoError(t, ada.reportPlayer(bob.account(), supportv1.ReportReason_REPORT_REASON_CHEATING, "", ""))
	assert.Equal(t, connect.CodeAlreadyExists,
		connect.CodeOf(ada.reportPlayer(bob.account(), supportv1.ReportReason_REPORT_REASON_BAD_NAME, "again", "")))
	require.NoError(t, ada.reportPlayer(cyd.account(), supportv1.ReportReason_REPORT_REASON_CHEATING, "", ""), "another player")

	assert.Len(t, game.reports(t), 2)
}

func TestADeletedAccountTakesTheReportsItFiledAndTheReportsAboutIt(t *testing.T) {
	game := startGame(t)
	leaving := game.namedPlayer(t, "google-ada", "Ada_L")
	bob := game.namedPlayer(t, "google-bob", "Bob")
	cyd := game.namedPlayer(t, "google-cyd", "Cyd")
	require.NoError(t, leaving.reportPlayer(bob.account(), supportv1.ReportReason_REPORT_REASON_CHEATING, "", ""))
	require.NoError(t, bob.reportPlayer(leaving.account(), supportv1.ReportReason_REPORT_REASON_BAD_NAME, "", ""))
	require.NoError(t, cyd.reportPlayer(bob.account(), supportv1.ReportReason_REPORT_REASON_OTHER, "kept", ""))

	deletion := connect.NewRequest(&authv1.DeleteAccountRequest{})
	leaving.send(deletion.Header())
	_, err := authv1connect.NewAuthServiceClient(http.DefaultClient, game.baseURL).DeleteAccount(t.Context(), deletion)
	require.NoError(t, err)

	var reports []*supportv1.Report
	require.Eventually(t, func() bool {
		reports = game.reports(t)
		return len(reports) == 1
	}, 5*time.Second, 20*time.Millisecond)
	assert.Equal(t, "kept", reports[0].GetText())
}

func rulesOf(t *testing.T, game gameStack) *supportv1.GetAttachmentRulesResponse {
	t.Helper()

	res, err := supportv1connect.NewSupportServiceClient(http.DefaultClient, game.baseURL).GetAttachmentRules(t.Context(),
		connect.NewRequest(&supportv1.GetAttachmentRulesRequest{}))
	require.NoError(t, err)
	return res.Msg
}

func TestAnAddressSendsTenNotesADayWhicheverAccountSendsThem(t *testing.T) {
	game := startGameWith(t, func(configs stackConfigs) {
		configs.support.RateLimiter.Burst = 100
		configs.support.RateLimiter.PerSecond = 100
	})
	first, second := game.newPlayer(t), game.newPlayer(t)

	for range 6 {
		require.NoError(t, first.suggest("an idea"))
	}
	for range 4 {
		require.NoError(t, second.reportBug("a bug", nil, ""))
	}

	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(second.suggest("one too many")))
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(first.reportBug("one too many", nil, "")))
	assert.Len(t, game.notes(t), 10)
}

func TestTheBucketTakesNoMoreBytesThanItsShelfAndThePageIsToldSo(t *testing.T) {
	game := startGameWith(t, func(configs stackConfigs) { configs.support.Attachments.MaxStoredBytes = 50 })
	guest := game.newPlayer(t)

	assert.Equal(t, uint64(50), rulesOf(t, game).GetMaxBytes())

	guest.uploaded(screenshot, recording)
	assert.Equal(t, uint64(20), rulesOf(t, game).GetMaxBytes(), "an unsent file takes its room too")

	_, err := guest.createUpload(recording)
	require.NoError(t, err)
	_, err = guest.createUpload(screenshot)
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

	_, err = guest.createUpload(file{contentType: "image/png", body: []byte("ab")})
	require.NoError(t, err)
	full := rulesOf(t, game)
	assert.Zero(t, full.GetMaxPerNote(), "a full bucket takes no file")
	assert.Zero(t, full.GetMaxBytes())
}
