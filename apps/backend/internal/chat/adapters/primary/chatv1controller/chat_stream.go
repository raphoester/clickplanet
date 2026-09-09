package chatv1controller

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain"
	"google.golang.org/protobuf/proto"
)

const MessageRoute = "/chat"

func EncodeMessage(message domain.ChatMessage) ([]byte, error) {
	return proto.Marshal(toProto(message))
}
