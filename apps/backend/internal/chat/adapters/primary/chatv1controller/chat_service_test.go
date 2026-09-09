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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain/chat_service"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/httpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
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

func startChatServer(t *testing.T, service chat_service.IService, blockedIPs []string) (*httptest.Server, *fakeClock) {
	t.Helper()

	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	limiter := ratelimit.New(ratelimit.Config{PerSecond: 1, Burst: 3}, clock)

	mux := http.NewServeMux()
	mux.Handle(chatv1connect.NewChatServiceHandler(
		NewChatService(service),
		connect.WithInterceptors(
			NewErrorInterceptor(nil),
			NewGuardInterceptor(limiter, blockedIPs),
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

// The whole chain has to agree on where the sender's address comes from: the
// domain stamps its tag from the context, which only the middleware fills in.
func TestSendMessagePassesTheUserAgentThrough(t *testing.T) {
	service := &stubService{}
	server, _ := startChatServer(t, service, nil)

	_, err := sendFrom(server, "1.2.3.4", "hello")
	require.NoError(t, err)

	require.Len(t, service.posted, 1)
	require.NotEmpty(t, service.posted[0].UserAgent, "the handler reads it off the request header")
}

// A refused message must say that it was refused and nothing else: the wrapped
// reason names the check that tripped, which is a hint worth withholding.
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
	server, _ := startChatServer(t, service, []string{"9.9.9.9"})

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

// The websocket carries the same message type the history RPC returns, so a
// client decodes one shape either way and deduplicates the overlap on id.
func TestEncodeMessageProducesABareChatMessage(t *testing.T) {
	bin, err := EncodeMessage(domain.ChatMessage{
		ID:         "message-1",
		SentAt:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		AuthorName: "Bob",
		AuthorTag:  "a1b2c3",
		CountryID:  "fr",
		Text:       "hello planet",
	})
	require.NoError(t, err)

	var decoded chatv1.ChatMessage
	require.NoError(t, proto.Unmarshal(bin, &decoded))
	require.Equal(t, "message-1", decoded.GetId())
	require.Equal(t, "hello planet", decoded.GetText())
	require.Equal(t, "a1b2c3", decoded.GetAuthorTag())
}

func sendOnce(server *httptest.Server, ip string) error {
	_, err := sendFrom(server, ip, "hello")
	return err
}

var _ chat_service.IService = (*stubService)(nil)
