// Package chatmessage puts a message and its reactions on the wire, and reads a reaction off it.
package chatmessage

import (
	"fmt"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

func Encode(message messages.Message) *chatv1.ChatMessage {
	return &chatv1.ChatMessage{
		Id:           string(message.ID),
		SentAtUnixMs: message.SentAt.UnixMilli(),
		AuthorName:   message.AuthorName,
		AuthorTag:    message.AuthorTag,
		AuthorAdmin:  message.AuthorAdmin,
		CountryId:    message.CountryID,
		Text:         message.Text,
		Reactions:    EncodeCounts(message.Reactions),
	}
}

func EncodeCounts(counts []messages.Count) []*chatv1.ReactionCount {
	encoded := make([]*chatv1.ReactionCount, 0, len(counts))
	for _, count := range counts {
		encoded = append(encoded, &chatv1.ReactionCount{
			Reaction: chatv1.Reaction(count.Reaction),
			Count:    uint32(count.Count), //nolint:gosec // a count of reactors, never negative.
			Mine:     count.Mine,
		})
	}
	return encoded
}

// Reaction is the wire's reaction as the domain's, or messages.ErrInvalidReaction for one the proto does not name.
func Reaction(reaction chatv1.Reaction) (messages.Reaction, error) {
	if _, named := chatv1.Reaction_name[int32(reaction)]; !named || reaction == chatv1.Reaction_REACTION_UNSPECIFIED {
		return 0, fmt.Errorf("%w: %d", messages.ErrInvalidReaction, reaction)
	}
	return messages.Reaction(reaction), nil
}
