package chatv1controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/mark_seen_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/react_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/send_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/usecases/react_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/usecases/mark_seen_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/stretchr/testify/require"
)

type stubSender struct {
	sent     int
	accounts []messages.AccountID
	err      error
}

func (s *stubSender) Execute(_ context.Context, in send_message_usecase.In) (messages.Message, error) {
	s.sent++
	s.accounts = append(s.accounts, in.Account)
	return messages.Message{}, s.err
}

type emptyHistory struct{}

func (emptyHistory) History(context.Context, messages.AccountID, messages.Room) (*chatv1.GetHistoryResponse, error) {
	return &chatv1.GetHistoryResponse{}, nil
}

type stubSubscriber struct {
	err error
}

func (s stubSubscriber) Subscribe(context.Context) (<-chan feed.Update, error) {
	return make(chan feed.Update), s.err
}

type stubReactor struct {
	reacted int
}

func (s *stubReactor) Execute(context.Context, react_usecase.In) (react_usecase.Out, error) {
	s.reacted++
	return react_usecase.Out{}, nil
}

type stubSeen struct {
	marked int
}

func (s *stubSeen) Execute(context.Context, mark_seen_usecase.In) error {
	s.marked++
	return nil
}

func startChatServer(
	t *testing.T,
	sender *stubSender,
	subscriber stubSubscriber,
	blockedIPs []string,
) (*httptest.Server, *cptime.FixedClock) {
	t.Helper()

	return chatServer{sender: sender, subscriber: subscriber, blockedIPs: blockedIPs}.start(t)
}

// The chain module.go mounts, in its order, before the stubbed use cases.
type chatServer struct {
	sender     *stubSender
	reactor    *stubReactor
	seen       *stubSeen
	subscriber stubSubscriber
	blockedIPs []string
	vpn        *cpipblock.Blocklist
}

func (c chatServer) start(t *testing.T) (*httptest.Server, *cptime.FixedClock) {
	t.Helper()

	if c.reactor == nil {
		c.reactor = &stubReactor{}
	}
	if c.seen == nil {
		c.seen = &stubSeen{}
	}

	clock := cptime.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	limiter := cpratelimit.New("test", cpratelimit.Config{PerSecond: 1, Burst: 3}, clock)

	blocklist, err := cpipblock.NewDenyList(c.blockedIPs)
	require.NoError(t, err)

	_, public := cpsession.TestKeyPair()
	verifier, err := cpsession.NewVerifier(public)
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.Handle(chatv1connect.NewChatServiceHandler(
		ChatService{
			SendMessageHandler: send_message_handler.New(c.sender),
			GetHistoryHandler:  get_history_handler.New(emptyHistory{}, messages.NewRooms(messages.FakeFlags{})),
			ListenForEventsHandler: listen_for_events_handler.New(listen_for_events_usecase.New(
				c.subscriber, messages.NewRooms(messages.FakeFlags{}), listen_for_events_usecase.DefaultHeartbeat)),
			ReactHandler:    react_handler.New(c.reactor),
			MarkSeenHandler: mark_seen_handler.New(c.seen),
		},
		connect.WithInterceptors(
			cpconnect.NewErrorInterceptor(nil, nil),
			NewBlocklistInterceptor(blocklist),
			NewVPNBlockInterceptor(c.vpn, prometheus.NewRegistry()),
			NewRateLimitInterceptor(limiter),
			NewSessionInterceptor(keyVerifier{verifier: verifier}, clock),
		),
	))

	server := httptest.NewServer(cphttpserver.IPReaderMiddleware(mux))
	t.Cleanup(server.Close)

	return server, clock
}

func vendoredVPNLists(t *testing.T) *cpipblock.Blocklist {
	t.Helper()

	lists := cpipblock.New(cpipblock.Config{Enabled: true})
	require.NoError(t, lists.Load())

	return lists
}

func sendOnce(server *httptest.Server, ip string) error {
	return sendWithToken(server, ip, "")
}

func sendWithToken(server *httptest.Server, ip string, token string) error {
	req := connect.NewRequest(&chatv1.SendMessageRequest{
		AuthorId:  "some-uuid",
		CountryId: "fr",
		Body:      &chatv1.SendMessageRequest_Text{Text: "hello"},
	})
	req.Header().Set("X-Real-IP", ip)
	if token != "" {
		req.Header().Set(cpconnect.SessionHeader, token)
	}

	_, err := chatv1connect.NewChatServiceClient(server.Client(), server.URL).
		SendMessage(context.Background(), req)

	return err
}

func reactOnce(server *httptest.Server, ip string) error {
	req := connect.NewRequest(&chatv1.ReactRequest{MessageId: "message-1", Reaction: chatv1.Reaction_REACTION_CLOWN, On: true})
	req.Header().Set("X-Real-IP", ip)

	_, err := chatv1connect.NewChatServiceClient(server.Client(), server.URL).React(context.Background(), req)

	return err //nolint:wrapcheck // the server's answer, as the caller reads it.
}

func readHistoryOnce(server *httptest.Server, ip string) error {
	req := connect.NewRequest(&chatv1.GetHistoryRequest{})
	req.Header().Set("X-Real-IP", ip)

	_, err := chatv1connect.NewChatServiceClient(server.Client(), server.URL).GetHistory(context.Background(), req)

	return err //nolint:wrapcheck // the server's answer, as the caller reads it.
}

func markSeenOnce(server *httptest.Server, ip string) error {
	req := connect.NewRequest(&chatv1.MarkSeenRequest{SeenUntilUnixMs: 1})
	req.Header().Set("X-Real-IP", ip)

	_, err := chatv1connect.NewChatServiceClient(server.Client(), server.URL).MarkSeen(context.Background(), req)

	return err //nolint:wrapcheck // the server's answer, as the caller reads it.
}

func TestAnUnexpectedFailureIsAnsweredAsAnInternalError(t *testing.T) {
	server, _ := startChatServer(t, &stubSender{err: errors.New("disk on fire")}, stubSubscriber{}, nil)

	err := sendOnce(server, "1.2.3.4")

	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "disk on fire")
}

func TestAFailedSubscriptionStaysInternalAndDoesNotLeakTheCause(t *testing.T) {
	server, _ := startChatServer(t, &stubSender{}, stubSubscriber{err: errors.New("disk on fire")}, nil)

	stream, err := chatv1connect.NewChatServiceClient(server.Client(), server.URL).
		ListenForEvents(t.Context(), connect.NewRequest(&chatv1.ListenForEventsRequest{}))
	if stream != nil {
		t.Cleanup(func() { _ = stream.Close() })
	}
	if err == nil {
		require.False(t, stream.Receive())
		err = stream.Err()
	}

	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "disk on fire")
}

func TestThrottleRefusesAFloodPerIP(t *testing.T) {
	sender := &stubSender{}
	server, clock := startChatServer(t, sender, stubSubscriber{}, nil)

	for i := range 3 {
		require.NoErrorf(t, sendOnce(server, "1.2.3.4"), "message %d should be allowed", i)
	}

	err := sendOnce(server, "1.2.3.4")
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	require.Equal(t, 3, sender.sent, "a refused message never reaches the use case")

	require.NoError(t, sendOnce(server, "5.6.7.8"), "another address has its own allowance")

	clock.Advance(time.Second)
	require.NoError(t, sendOnce(server, "1.2.3.4"), "a second later the bucket has a token again")
}

func TestABlockedSenderIsRefused(t *testing.T) {
	sender := &stubSender{}
	server, _ := startChatServer(t, sender, stubSubscriber{}, []string{"9.9.9.9/32"})

	err := sendOnce(server, "9.9.9.9")
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	require.Zero(t, sender.sent)

	require.NoError(t, sendOnce(server, "1.2.3.4"), "everyone else is unaffected")
}

// The raid of November 6, 2026 posted slurs from this /64, Cloudflare WARP's.
const warpAddress = "2a09:bac5:d594:2682::1"

func TestAVPNAddressReadsTheChatAndMayNotSpeakInIt(t *testing.T) {
	sender, reactor, seen := &stubSender{}, &stubReactor{}, &stubSeen{}
	server, _ := chatServer{sender: sender, reactor: reactor, seen: seen, vpn: vendoredVPNLists(t)}.start(t)

	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(sendOnce(server, warpAddress)))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(reactOnce(server, warpAddress)))
	require.Zero(t, sender.sent, "a refused post never reaches the use case")
	require.Zero(t, reactor.reacted, "a refused reaction never reaches the use case")

	require.NoError(t, readHistoryOnce(server, warpAddress), "a VPN still reads the history")
	require.NoError(t, markSeenOnce(server, warpAddress), "and keeps where it read to")
	require.Equal(t, 1, seen.marked)
}

func TestAnAddressOnNoVPNListSpeaksAsBefore(t *testing.T) {
	sender, reactor := &stubSender{}, &stubReactor{}
	server, _ := chatServer{sender: sender, reactor: reactor, vpn: vendoredVPNLists(t)}.start(t)

	require.NoError(t, sendOnce(server, "8.8.8.8"))
	require.NoError(t, reactOnce(server, "8.8.8.8"))
	require.Equal(t, 1, sender.sent)
	require.Equal(t, 1, reactor.reacted)
}

func TestAVPNRefusalSpendsNoToken(t *testing.T) {
	server, _ := chatServer{sender: &stubSender{}, vpn: vendoredVPNLists(t)}.start(t)

	for i := range 5 {
		require.Equalf(t, connect.CodePermissionDenied, connect.CodeOf(sendOnce(server, warpAddress)),
			"post %d is refused for its address, never throttled: the lists sit outside the limiter", i)
	}
}

func TestAVPNRefusalIsTheBlocklistsAnswer(t *testing.T) {
	server, _ := chatServer{sender: &stubSender{}, blockedIPs: []string{"9.9.9.9/32"}, vpn: vendoredVPNLists(t)}.start(t)

	var denied, vpn *connect.Error
	require.ErrorAs(t, sendOnce(server, "9.9.9.9"), &denied)
	require.ErrorAs(t, sendOnce(server, warpAddress), &vpn)

	require.Equal(t, denied.Code(), vpn.Code())
	require.Equal(t, denied.Message(), vpn.Message(), "a sender is told it is refused, not which list it is on")
	require.Empty(t, vpn.Details(), "no MuteRefusal: the client tells a block from a mute by it")
}

func TestAValidTokenNamesTheSendersAccountAndABadOneIsAGuest(t *testing.T) {
	sender := &stubSender{}
	server, clock := startChatServer(t, sender, stubSubscriber{}, nil)
	secret, _ := cpsession.TestKeyPair()
	signer, err := cpsession.NewSigner(cpsession.SignerConfig{Enabled: true, Secret: secret, TTL: time.Hour})
	require.NoError(t, err)
	ada := cpsession.AccountID{15: 1}
	token, err := signer.Mint("1.2.3.4", cpsession.Holder{Account: ada, Attested: true}, clock.Now())
	require.NoError(t, err)
	identity, err := signer.Mint("1.2.3.4", cpsession.Holder{Account: ada}, clock.Now())
	require.NoError(t, err)

	require.NoError(t, sendWithToken(server, "1.2.3.4", token.Value))
	require.NoError(t, sendWithToken(server, "1.2.3.4", "forged"), "a bad token is a guest, never a refusal")
	require.NoError(t, sendWithToken(server, "5.6.7.8", token.Value), "a token from another address is a guest")
	require.NoError(t, sendOnce(server, "1.2.3.4"))
	clock.Advance(time.Minute)
	require.NoError(t, sendWithToken(server, "1.2.3.4", identity.Value), "a post needs the Turnstile check, not a name")

	require.Equal(t, []messages.AccountID{ada, cpsession.NoAccount, cpsession.NoAccount, cpsession.NoAccount, cpsession.NoAccount},
		sender.accounts)
}

type keyVerifier struct {
	verifier *cpsession.Verifier
}

func (k keyVerifier) Verify(_ context.Context, token string, ip string, now time.Time) (*cpsession.Claims, error) {
	return k.verifier.Verify(token, ip, now) //nolint:wrapcheck // the key's own verdict.
}
