package chatv1controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/get_history_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/send_message_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/stretchr/testify/require"
)

// These tests are about the interceptor chain; each procedure's mapping is tested in its handler package.

type stubSender struct {
	sent int
	err  error
}

func (s *stubSender) Execute(context.Context, send_message_usecase.In) (messages.Message, error) {
	s.sent++
	return messages.Message{}, s.err
}

type emptyHistory struct{}

func (emptyHistory) Execute(context.Context) []messages.Message { return nil }

type stubSubscriber struct {
	err error
}

type stubBans map[string]bool

func (s stubBans) Banned(tag string) bool { return s[tag] }

const testSalt = "pepper"

func (s stubSubscriber) Subscribe(context.Context) (<-chan messages.Event, error) {
	return make(chan messages.Event), s.err
}

func startChatServer(
	t *testing.T,
	sender *stubSender,
	subscriber stubSubscriber,
	blockedIPs []string,
	bannedTags ...string,
) (*httptest.Server, *cptime.FixedClock) {
	t.Helper()

	clock := cptime.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	limiter := cpratelimit.New("test", cpratelimit.Config{PerSecond: 1, Burst: 3}, clock)

	blocklist, err := cpipblock.NewDenyList(blockedIPs)
	require.NoError(t, err)

	banned := stubBans{}
	for _, tag := range bannedTags {
		banned[tag] = true
	}

	mux := http.NewServeMux()
	mux.Handle(chatv1connect.NewChatServiceHandler(
		ChatService{
			SendMessageHandler: send_message_handler.New(sender),
			GetHistoryHandler:  get_history_handler.New(emptyHistory{}),
			ListenForEventsHandler: listen_for_events_handler.New(
				listen_for_events_usecase.New(subscriber, listen_for_events_usecase.DefaultHeartbeat)),
		},
		connect.WithInterceptors(
			cpconnect.NewErrorInterceptor(nil, nil),
			NewBlocklistInterceptor(blocklist),
			NewBanInterceptor(banned, messages.NewTagger(testSalt)),
			NewRateLimitInterceptor(limiter),
		),
	))

	server := httptest.NewServer(cphttpserver.IPReaderMiddleware(mux))
	t.Cleanup(server.Close)

	return server, clock
}

func sendOnce(server *httptest.Server, ip string) error {
	req := connect.NewRequest(&chatv1.SendMessageRequest{
		AuthorName: "Bob",
		AuthorId:   "some-uuid",
		CountryId:  "fr",
		Text:       "hello",
	})
	req.Header().Set("X-Real-IP", ip)

	_, err := chatv1connect.NewChatServiceClient(server.Client(), server.URL).
		SendMessage(context.Background(), req)

	return err
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

func TestABannedMemberIsRefusedAndNeverSpendsAToken(t *testing.T) {
	sender := &stubSender{}
	banned := messages.NewTagger(testSalt).Of("9.9.9.9")
	server, _ := startChatServer(t, sender, stubSubscriber{}, nil, banned)

	for i := range 5 {
		err := sendOnce(server, "9.9.9.9")
		require.Equalf(t, connect.CodePermissionDenied, connect.CodeOf(err),
			"message %d: a ban outside the limiter never turns into a throttle", i)
	}

	require.Zero(t, sender.sent)
	require.NoError(t, sendOnce(server, "1.2.3.4"), "everyone else is unaffected")
}

func TestABannedMemberCannotRenumberOutOfItsBan(t *testing.T) {
	banned := messages.NewTagger(testSalt).Of("2001:db8:1:2::1")
	server, _ := startChatServer(t, &stubSender{}, stubSubscriber{}, nil, banned)

	err := sendOnce(server, "2001:db8:1:2:dead:beef:0:9")

	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestABannedMemberCanStillReadTheChat(t *testing.T) {
	banned := messages.NewTagger(testSalt).Of("9.9.9.9")
	server, _ := startChatServer(t, &stubSender{}, stubSubscriber{}, nil, banned)

	req := connect.NewRequest(&chatv1.GetHistoryRequest{})
	req.Header().Set("X-Real-IP", "9.9.9.9")

	_, err := chatv1connect.NewChatServiceClient(server.Client(), server.URL).
		GetHistory(t.Context(), req)

	require.NoError(t, err)
}
