package chatv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain"
	"google.golang.org/protobuf/proto"
)

// MessageRoute is where the chat stream is served, under the app's /ws prefix.
// It is a route of its own rather than a second payload on the tile stream:
// frames carry a bare protobuf message with no type tag, so a client reading
// /ws/listen cannot tell one message type from another.
const MessageRoute = "/chat"

// EncodeMessage is what the websocket fanout sends — the same ChatMessage the
// history RPC returns, so a client decodes one type either way and deduplicates
// the overlap on id.
func EncodeMessage(message domain.ChatMessage) ([]byte, error) {
	return proto.Marshal(toProto(message))
}
