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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/stretchr/testify/require"
)

// These tests are about the interceptor chain; each procedure's mapping is tested in its handler package.

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

func (emptyHistory) Execute(context.Context) []messages.Message { return nil }

type stubSubscriber struct {
	err error
}

func (s stubSubscriber) Subscribe(context.Context) (<-chan messages.Message, error) {
	return make(chan messages.Message), s.err
}

func startChatServer(
	t *testing.T,
	sender *stubSender,
	subscriber stubSubscriber,
	blockedIPs []string,
) (*httptest.Server, *cptime.FixedClock) {
	t.Helper()

	clock := cptime.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	limiter := cpratelimit.New("test", cpratelimit.Config{PerSecond: 1, Burst: 3}, clock)

	blocklist, err := cpipblock.NewDenyList(blockedIPs)
	require.NoError(t, err)

	_, public := cpsession.TestKeyPair()
	verifier, err := cpsession.NewVerifier(public)
	require.NoError(t, err)

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
			NewRateLimitInterceptor(limiter),
			NewSessionInterceptor(verifier, clock),
		),
	))

	server := httptest.NewServer(cphttpserver.IPReaderMiddleware(mux))
	t.Cleanup(server.Close)

	return server, clock
}

func sendOnce(server *httptest.Server, ip string) error {
	return sendWithToken(server, ip, "")
}

func sendWithToken(server *httptest.Server, ip string, token string) error {
	req := connect.NewRequest(&chatv1.SendMessageRequest{
		AuthorName: "Bob",
		AuthorId:   "some-uuid",
		CountryId:  "fr",
		Text:       "hello",
	})
	req.Header().Set("X-Real-IP", ip)
	if token != "" {
		req.Header().Set(cpconnect.SessionHeader, token)
	}

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

func TestAValidTokenNamesTheSendersAccountAndABadOneIsAGuest(t *testing.T) {
	sender := &stubSender{}
	server, clock := startChatServer(t, sender, stubSubscriber{}, nil)
	secret, _ := cpsession.TestKeyPair()
	signer, err := cpsession.NewSigner(cpsession.SignerConfig{Enabled: true, Secret: secret, TTL: time.Hour})
	require.NoError(t, err)
	ada := cpsession.AccountID{15: 1}
	token, err := signer.Mint("1.2.3.4", cpsession.Holder{Account: ada}, clock.Now())
	require.NoError(t, err)

	require.NoError(t, sendWithToken(server, "1.2.3.4", token.Value))
	require.NoError(t, sendWithToken(server, "1.2.3.4", "forged"), "a bad token is a guest, never a refusal")
	require.NoError(t, sendWithToken(server, "5.6.7.8", token.Value), "a token from another address is a guest")
	require.NoError(t, sendOnce(server, "1.2.3.4"))

	require.Equal(t, []messages.AccountID{ada, cpsession.NoAccount, cpsession.NoAccount, cpsession.NoAccount}, sender.accounts)
}
