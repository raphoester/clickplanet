package chatv1controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/domain/chat_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/httpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/ipblock"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/ratelimit"
	"github.com/stretchr/testify/require"
)

type stubService struct {
	posted  []chat_service.PostRequest
	err     error
	history []domain.ChatMessage
}

func (s *stubService) Post(_ context.Context, req chat_service.PostRequest) (domain.ChatMessage, error) {
	s.posted = append(s.posted, req)
	if s.err != nil {
		return domain.ChatMessage{}, s.err
	}

	return domain.ChatMessage{
		ID:         "message-1",
		SentAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AuthorName: req.AuthorName,
		AuthorTag:  "a1b2c3",
		CountryID:  req.CountryID,
		Text:       req.Text,
	}, nil
}

func (s *stubService) History(context.Context) []domain.ChatMessage {
	return s.history
}

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) advance(d time.Duration) { c.now = c.now.Add(d) }

type stubSubscriber struct {
	messages chan domain.ChatMessage
	err      error
}

func (s stubSubscriber) Subscribe(context.Context) (<-chan domain.ChatMessage, error) {
	return s.messages, s.err
}

func startChatServer(t *testing.T, service chat_service.IService, blockedIPs []string) (*httptest.Server, *fakeClock) {
	t.Helper()
	return startChatServerWith(t, service, blockedIPs, stubSubscriber{}, DefaultHeartbeat)
}

func startChatServerWith(
	t *testing.T,
	service chat_service.IService,
	blockedIPs []string,
	subscriber MessagesSubscriber,
	heartbeat time.Duration,
) (*httptest.Server, *fakeClock) {
	t.Helper()

	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	limiter := ratelimit.New(ratelimit.Config{PerSecond: 1, Burst: 3}, clock)

	blocklist, err := ipblock.NewDenyList(blockedIPs)
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.Handle(chatv1connect.NewChatServiceHandler(
		NewChatService(service, subscriber, heartbeat),
		connect.WithInterceptors(
			NewErrorInterceptor(nil),
			NewBlocklistInterceptor(blocklist),
			NewRateLimitInterceptor(limiter),
		),
	))

	server := httptest.NewServer(httpserver.IPReaderMiddleware(mux))
	t.Cleanup(server.Close)

	return server, clock
}

func sendFrom(server *httptest.Server, ip string, text string) (*connect.Response[chatv1.SendMessageResponse], error) {
	req := connect.NewRequest(&chatv1.SendMessageRequest{
		AuthorName: "Bob",
		AuthorId:   "some-uuid",
		CountryId:  "fr",
		Text:       text,
	})
	req.Header().Set("X-Real-IP", ip)

	return chatv1connect.NewChatServiceClient(server.Client(), server.URL).
		SendMessage(context.Background(), req)
}

func TestListenForEvents(t *testing.T) {
	// The call blocks until the first frame, so subscriptions are seeded before it.
	listen := func(
		t *testing.T,
		subscriber MessagesSubscriber,
		heartbeat time.Duration,
	) (*connect.ServerStreamForClient[chatv1.ChatEvent], error) {
		t.Helper()

		server, _ := startChatServerWith(t, &stubService{}, nil, subscriber, heartbeat)

		stream, err := chatv1connect.NewChatServiceClient(server.Client(), server.URL).
			ListenForEvents(context.Background(), connect.NewRequest(&chatv1.ListenForEventsRequest{}))

		// Closing it releases the handler, which is still parked on its
		// subscription; httptest.Server.Close blocks forever otherwise.
		if stream != nil {
			t.Cleanup(func() { _ = stream.Close() })
		}

		return stream, err
	}

	t.Run("carries a message to the caller", func(t *testing.T) {
		messages := make(chan domain.ChatMessage, 1)
		messages <- domain.ChatMessage{
			ID:         "message-1",
			SentAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			AuthorName: "Bob",
			AuthorTag:  "a1b2c3",
			CountryID:  "fr",
			Text:       "hello",
		}

		stream, err := listen(t, stubSubscriber{messages: messages}, DefaultHeartbeat)
		require.NoError(t, err)

		require.True(t, stream.Receive())
		posted := stream.Msg().GetMessage()
		require.NotNil(t, posted, "a message arrives as the message case")
		require.Equal(t, "message-1", posted.GetId())
		require.Equal(t, "Bob", posted.GetAuthorName())
		require.Equal(t, "hello", posted.GetText())
	})

	t.Run("keeps a silent stream alive with heartbeats", func(t *testing.T) {
		// A quiet chat is the normal case, so every frame here is a heartbeat.
		stream, err := listen(t, stubSubscriber{messages: make(chan domain.ChatMessage)}, 10*time.Millisecond)
		require.NoError(t, err)

		for i := range 3 {
			require.Truef(t, stream.Receive(), "heartbeat %d never arrived", i)
			require.NotNil(t, stream.Msg().GetHeartbeat(), "frame %d is not a heartbeat", i)
			require.Nil(t, stream.Msg().GetMessage())
		}
	})

	t.Run("ends when the subscription closes", func(t *testing.T) {
		messages := make(chan domain.ChatMessage)
		close(messages)

		stream, err := listen(t, stubSubscriber{messages: messages}, DefaultHeartbeat)
		require.NoError(t, err)

		require.False(t, stream.Receive())
		require.NoError(t, stream.Err())
	})

	t.Run("a failed subscription stays internal and does not leak the cause", func(t *testing.T) {
		stream, err := listen(t, stubSubscriber{err: errors.New("disk on fire")}, DefaultHeartbeat)
		if err == nil {
			require.False(t, stream.Receive())
			err = stream.Err()
		}

		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		require.NotContains(t, err.Error(), "disk on fire")
	})
}

func TestSendMessageReturnsTheStoredMessage(t *testing.T) {
	service := &stubService{}
	server, _ := startChatServer(t, service, nil)

	res, err := sendFrom(server, "1.2.3.4", "hello planet")
	require.NoError(t, err)

	message := res.Msg.GetMessage()
	require.Equal(t, "message-1", message.GetId())
	require.Equal(t, "Bob", message.GetAuthorName())
	require.Equal(t, "a1b2c3", message.GetAuthorTag())
	require.Equal(t, "hello planet", message.GetText())
	require.Equal(t, int64(1704067200000), message.GetSentAtUnixMs())
}

func TestSendMessagePassesTheUserAgentThrough(t *testing.T) {
	service := &stubService{}
	server, _ := startChatServer(t, service, nil)

	_, err := sendFrom(server, "1.2.3.4", "hello")
	require.NoError(t, err)

	require.Len(t, service.posted, 1)
	require.NotEmpty(t, service.posted[0].UserAgent, "the handler reads it off the request header")
}

func TestARefusedMessageDoesNotLeakWhyItWasRefused(t *testing.T) {
	service := &stubService{
		err: fmt.Errorf("%w: text: longer than 280 characters", domain.ErrInvalidMessage),
	}
	server, _ := startChatServer(t, service, nil)

	_, err := sendFrom(server, "1.2.3.4", "whatever")

	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "280")
	require.NotContains(t, err.Error(), "text:")
}

func TestAnUnexpectedFailureIsAnsweredAsAnInternalError(t *testing.T) {
	service := &stubService{err: errors.New("disk on fire")}
	server, _ := startChatServer(t, service, nil)

	_, err := sendFrom(server, "1.2.3.4", "hello")

	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	require.NotContains(t, err.Error(), "disk on fire")
}

func TestThrottleRefusesAFloodPerIP(t *testing.T) {
	service := &stubService{}
	server, clock := startChatServer(t, service, nil)

	for i := range 3 {
		require.NoErrorf(t, sendOnce(server, "1.2.3.4"), "message %d should be allowed", i)
	}

	err := sendOnce(server, "1.2.3.4")
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	require.Len(t, service.posted, 3, "a refused message never reaches the domain")

	require.NoError(t, sendOnce(server, "5.6.7.8"), "another address has its own allowance")

	clock.advance(time.Second)
	require.NoError(t, sendOnce(server, "1.2.3.4"), "a second later the bucket has a token again")
}

func TestABlockedSenderIsRefused(t *testing.T) {
	service := &stubService{}
	server, _ := startChatServer(t, service, []string{"9.9.9.9/32"})

	err := sendOnce(server, "9.9.9.9")
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	require.Empty(t, service.posted)

	require.NoError(t, sendOnce(server, "1.2.3.4"), "everyone else is unaffected")
}

func TestGetHistoryIsNeverCached(t *testing.T) {
	service := &stubService{history: []domain.ChatMessage{{ID: "message-1", Text: "hello"}}}
	server, _ := startChatServer(t, service, nil)

	res, err := chatv1connect.NewChatServiceClient(server.Client(), server.URL).
		GetHistory(context.Background(), connect.NewRequest(&chatv1.GetHistoryRequest{}))
	require.NoError(t, err)

	require.Len(t, res.Msg.GetMessages(), 1)
	require.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}

func sendOnce(server *httptest.Server, ip string) error {
	_, err := sendFrom(server, ip, "hello")
	return err
}

var _ chat_service.IService = (*stubService)(nil)
