package e2e_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1/missionsv1connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1/playerv1connect"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1/supportv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
)

type gamer struct {
	t      *testing.T
	stack  gameStack
	cookie string
	token  string

	adminCookie string
}

func (s gameStack) newPlayer(t *testing.T) *gamer {
	t.Helper()

	req := connect.NewRequest(&authv1.CreateSessionRequest{AttestationToken: "unused"})
	req.Header().Set("X-Real-IP", callerIP)
	res, err := authv1connect.NewAuthServiceClient(http.DefaultClient, s.baseURL).CreateSession(t.Context(), req)
	require.NoError(t, err)
	cookie, err := http.ParseSetCookie(res.Header().Get("Set-Cookie"))
	require.NoError(t, err)

	return &gamer{t: t, stack: s, cookie: cookie.Name + "=" + cookie.Value, token: res.Msg.GetToken()}
}

func (s gameStack) namedPlayer(t *testing.T, subject, name string) *gamer {
	t.Helper()

	named := s.newPlayer(t)
	named.link(subject)
	require.NoError(t, named.setName(name))
	return named
}

func (p *gamer) send(header http.Header) {
	header.Set("X-Real-IP", callerIP)
	header.Set(cpconnect.SessionHeader, p.token)
	header.Set("Cookie", p.cookie)
}

func (p *gamer) auth() authv1connect.AuthServiceClient {
	return authv1connect.NewAuthServiceClient(http.DefaultClient, p.stack.baseURL)
}

func (p *gamer) link(subject string) {
	p.t.Helper()

	start := connect.NewRequest(&authv1.StartSignInRequest{
		Provider: authv1.Provider_PROVIDER_GOOGLE, Intent: authv1.SignInIntent_SIGN_IN_INTENT_LINK,
	})
	p.send(start.Header())
	started, err := p.auth().StartSignIn(p.t.Context(), start)
	require.NoError(p.t, err)
	flow, err := http.ParseSetCookie(started.Header().Get("Set-Cookie"))
	require.NoError(p.t, err)
	authorization, err := url.Parse(started.Msg.GetAuthorizationUrl())
	require.NoError(p.t, err)

	p.stack.fakes.Google.Grant(subject, auth.ClaimOf(subject, "", false))
	complete := connect.NewRequest(&authv1.CompleteSignInRequest{Code: subject, State: authorization.Query().Get("state")})
	p.send(complete.Header())
	complete.Header().Set("Cookie", p.cookie+"; "+flow.Name+"="+flow.Value)
	completed, err := p.auth().CompleteSignIn(p.t.Context(), complete)
	require.NoError(p.t, err)
	require.Equal(p.t, authv1.SignInOutcome_SIGN_IN_OUTCOME_LINKED, completed.Msg.GetOutcome())

	for _, line := range completed.Header().Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(line)
		require.NoError(p.t, err)
		if cookie.MaxAge >= 0 {
			p.cookie = cookie.Name + "=" + cookie.Value
		}
	}

	mint := connect.NewRequest(&authv1.CreateSessionRequest{AttestationToken: "unused"})
	p.send(mint.Header())
	minted, err := p.auth().CreateSession(p.t.Context(), mint)
	require.NoError(p.t, err)
	p.token = minted.Msg.GetToken()
}

func (p *gamer) me() *authv1.GetMeResponse {
	p.t.Helper()

	req := connect.NewRequest(&authv1.GetMeRequest{})
	p.send(req.Header())
	res, err := p.auth().GetMe(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg
}

func (p *gamer) deleteAccount() {
	p.t.Helper()

	req := connect.NewRequest(&authv1.DeleteAccountRequest{})
	p.send(req.Header())
	_, err := p.auth().DeleteAccount(p.t.Context(), req)
	require.NoError(p.t, err)
}

func (p *gamer) players() playerv1connect.PlayerServiceClient {
	return playerv1connect.NewPlayerServiceClient(http.DefaultClient, p.stack.baseURL)
}

func (p *gamer) setName(name string) error {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.SetNameRequest{Name: name})
	p.send(req.Header())
	if _, err := p.players().SetName(p.t.Context(), req); err != nil {
		return fmt.Errorf("SetName failed: %w", err)
	}
	return nil
}

func (p *gamer) profile() *playerv1.Profile {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.GetProfileRequest{})
	p.send(req.Header())
	res, err := p.players().GetProfile(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetProfile()
}

func (p *gamer) account() string {
	p.t.Helper()

	return p.profile().GetAccountId()
}

func (p *gamer) stats() *playerv1.Stats {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.GetStatsRequest{})
	p.send(req.Header())
	res, err := p.players().GetStats(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetStats()
}

func (p *gamer) track(id string) *playerv1.Track {
	p.t.Helper()

	req := connect.NewRequest(&playerv1.GetTitlesRequest{})
	p.send(req.Header())
	res, err := p.players().GetTitles(p.t.Context(), req)
	require.NoError(p.t, err)
	for _, track := range res.Msg.GetTracks() {
		if track.GetId() == id {
			return track
		}
	}
	return nil
}

func (p *gamer) clicks() planetv1connect.ClickServiceClient {
	return planetv1connect.NewClickServiceClient(http.DefaultClient, p.stack.baseURL)
}

func (p *gamer) click(tile uint32, country string) {
	p.t.Helper()

	held := connect.NewRequest(&planetv1.GetAllegianceRequest{})
	p.send(held.Header())
	allegiance, err := p.clicks().GetAllegiance(p.t.Context(), held)
	require.NoError(p.t, err)
	if allegiance.Msg.GetAllegiance() == nil {
		p.pledge(country)
	}

	req := connect.NewRequest(&planetv1.ClickRequest{TileId: tile, CountryId: country})
	p.send(req.Header())
	_, err = p.clicks().Click(p.t.Context(), req)
	require.NoError(p.t, err)
}

func (p *gamer) pledge(country string) {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.PledgeAllegianceRequest{CountryId: country})
	p.send(req.Header())
	_, err := p.clicks().PledgeAllegiance(p.t.Context(), req)
	require.NoError(p.t, err, "the web client pledges the flag clicked for when it holds none")
}

func (p *gamer) charges() *planetv1.ChargesHeld {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.GetChargesRequest{})
	p.send(req.Header())
	res, err := p.clicks().GetCharges(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetCharges()
}

func (p *gamer) missionDesk() missionsv1connect.MissionServiceClient {
	return missionsv1connect.NewMissionServiceClient(http.DefaultClient, p.stack.baseURL)
}

func (p *gamer) missions() []*missionsv1.Mission {
	p.t.Helper()

	req := connect.NewRequest(&missionsv1.GetMissionsRequest{})
	p.send(req.Header())
	res, err := p.missionDesk().GetMissions(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetMissions()
}

func (p *gamer) collect(mission string) *missionsv1.CollectMissionResponse {
	p.t.Helper()

	req := connect.NewRequest(&missionsv1.CollectMissionRequest{MissionId: mission})
	p.send(req.Header())
	res, err := p.missionDesk().CollectMission(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg
}

func (p *gamer) mySeason() *seasonsv1.GetMySeasonResponse {
	p.t.Helper()

	req := connect.NewRequest(&seasonsv1.GetMySeasonRequest{})
	p.send(req.Header())
	res, err := seasonsv1connect.NewSeasonServiceClient(http.DefaultClient, p.stack.baseURL).GetMySeason(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg
}

func (p *gamer) chat() chatv1connect.ChatServiceClient {
	return chatv1connect.NewChatServiceClient(http.DefaultClient, p.stack.baseURL)
}

func (p *gamer) post(text string) *chatv1.ChatMessage {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.SendMessageRequest{AuthorId: "browser-1", CountryId: "fr", Text: text})
	p.send(req.Header())
	res, err := p.chat().SendMessage(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg.GetMessage()
}

func (p *gamer) history() *chatv1.GetHistoryResponse {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.GetHistoryRequest{})
	p.send(req.Header())
	res, err := p.chat().GetHistory(p.t.Context(), req)
	require.NoError(p.t, err)
	return res.Msg
}

func (p *gamer) markSeen(at time.Time) {
	p.t.Helper()

	req := connect.NewRequest(&chatv1.MarkSeenRequest{SeenUntilUnixMs: at.UnixMilli()})
	p.send(req.Header())
	_, err := p.chat().MarkSeen(p.t.Context(), req)
	require.NoError(p.t, err)
}

func (p *gamer) support() supportv1connect.SupportServiceClient {
	return supportv1connect.NewSupportServiceClient(http.DefaultClient, p.stack.baseURL)
}

type file struct {
	contentType string
	body        []byte
}

func (p *gamer) uploaded(sent file) string {
	p.t.Helper()

	req := connect.NewRequest(&supportv1.CreateUploadRequest{ContentType: sent.contentType, SizeBytes: uint64(len(sent.body))})
	p.send(req.Header())
	upload, err := p.support().CreateUpload(p.t.Context(), req)
	require.NoError(p.t, err)

	put, err := http.NewRequestWithContext(p.t.Context(), http.MethodPut, upload.Msg.GetUrl(), bytes.NewReader(sent.body))
	require.NoError(p.t, err)
	put.Header.Set("Content-Type", sent.contentType)
	res, err := http.DefaultClient.Do(put)
	require.NoError(p.t, err)
	require.NoError(p.t, res.Body.Close())
	require.Equal(p.t, http.StatusOK, res.StatusCode)

	return upload.Msg.GetAttachmentId()
}

func (p *gamer) reportBug(text string, attachments ...string) {
	p.t.Helper()

	req := connect.NewRequest(&supportv1.ReportBugRequest{Text: text, AttachmentIds: attachments})
	p.send(req.Header())
	_, err := p.support().ReportBug(p.t.Context(), req)
	require.NoError(p.t, err)
}

func (s gameStack) noteOf(t *testing.T, text string) *supportv1.Note {
	t.Helper()

	res, err := supportv1connect.NewAdminServiceClient(http.DefaultClient, s.consoleURL).ListNotes(t.Context(),
		connect.NewRequest(&supportv1.ListNotesRequest{}))
	require.NoError(t, err)
	for _, note := range res.Msg.GetNotes() {
		if note.GetText() == text {
			return note
		}
	}
	return nil
}

func opened(t *testing.T, attachment *supportv1.NoteAttachment) file {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, attachment.GetUrl(), nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = res.Body.Close() }()
	require.Equal(t, http.StatusOK, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return file{contentType: res.Header.Get("Content-Type"), body: body}
}

func (p *gamer) becomeOperator() {
	p.t.Helper()

	_, err := p.stack.schema(p.t, "auth").On(p.t.Context()).ExecContext(p.t.Context(),
		`UPDATE accounts SET operator = true WHERE id = $1`, p.account())
	require.NoError(p.t, err)
}

func (p *gamer) desk() authv1connect.AdminServiceClient {
	return authv1connect.NewAdminServiceClient(http.DefaultClient, p.stack.adminURL)
}

func (p *gamer) adminSignIn(googleSubject string) {
	p.t.Helper()

	start := connect.NewRequest(&authv1.AdminStartSignInRequest{})
	start.Header().Set("Origin", adminOrigin)
	started, err := p.desk().StartSignIn(p.t.Context(), start)
	require.NoError(p.t, err)
	flow, err := http.ParseSetCookie(started.Header().Get("Set-Cookie"))
	require.NoError(p.t, err)
	authorization, err := url.Parse(started.Msg.GetAuthorizationUrl())
	require.NoError(p.t, err)

	p.stack.fakes.AdminGoogle.Grant(googleSubject, auth.ClaimOf(googleSubject, "", false))
	complete := connect.NewRequest(&authv1.AdminCompleteSignInRequest{
		Code: googleSubject, State: authorization.Query().Get("state"),
	})
	complete.Header().Set("Origin", adminOrigin)
	complete.Header().Set("Cookie", p.cookie+"; "+flow.Name+"="+flow.Value)
	completed, err := p.desk().CompleteSignIn(p.t.Context(), complete)
	require.NoError(p.t, err)

	for _, line := range completed.Header().Values("Set-Cookie") {
		cookie, err := http.ParseSetCookie(line)
		require.NoError(p.t, err)
		if cookie.Name == "cp_admin" {
			p.adminCookie = cookie.Name + "=" + cookie.Value
		}
	}
	require.NotEmpty(p.t, p.adminCookie, "a completed admin sign-in sets the admin session")
}

func (p *gamer) grantFromThePage(account string) (*planetv1.GrantChargesResponse, error) {
	p.t.Helper()

	req := connect.NewRequest(&planetv1.GrantChargesRequest{AccountId: account, Bomb: true})
	req.Header().Set("Origin", adminOrigin)
	req.Header().Set("Cookie", p.cookie+"; "+p.adminCookie)
	res, err := planetv1connect.NewAdminServiceClient(http.DefaultClient, p.stack.adminURL).GrantCharges(p.t.Context(), req)
	if err != nil {
		return nil, err //nolint:wrapcheck // the tests read the connect code.
	}
	return res.Msg, nil
}
