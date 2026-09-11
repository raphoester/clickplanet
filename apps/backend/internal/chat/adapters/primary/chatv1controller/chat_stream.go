package chatv1controller

import (
	"time"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/domain"
)

// Well under Cloudflare's ~125s idle cut, and cheap: a heartbeat is two bytes.
const DefaultHeartbeat = 30 * time.Second

func messageEvent(message domain.ChatMessage) *chatv1.ChatEvent {
	return &chatv1.ChatEvent{
		Event: &chatv1.ChatEvent_Message{Message: toProto(message)},
	}
}

func heartbeatEvent() *chatv1.ChatEvent {
	return &chatv1.ChatEvent{
		Event: &chatv1.ChatEvent_Heartbeat{Heartbeat: &chatv1.Heartbeat{}},
	}
}
