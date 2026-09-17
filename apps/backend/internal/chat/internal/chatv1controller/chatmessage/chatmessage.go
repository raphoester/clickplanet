// Package chatmessage puts a message on the wire; all three procedures send one.
package chatmessage

import (
	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

func Encode(message messages.Message) *chatv1.ChatMessage {
	return &chatv1.ChatMessage{
		Id:           message.ID,
		SentAtUnixMs: message.SentAt.UnixMilli(),
		AuthorName:   message.AuthorName,
		AuthorTag:    message.AuthorTag,
		AuthorAdmin:  message.AuthorAdmin,
		CountryId:    message.CountryID,
		Text:         message.Text,
	}
}
