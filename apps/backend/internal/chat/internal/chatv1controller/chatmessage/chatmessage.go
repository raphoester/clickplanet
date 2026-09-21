// Package chatmessage puts a message and its reactions on the wire, and reads a reaction off it.
package chatmessage

import (
	"fmt"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

// Encode is a message on the wire, with its reactions as the reader sees them: none on a message just sent.
func Encode(message messages.Message, counts []reactions.Count, version uint64) *chatv1.ChatMessage {
	return &chatv1.ChatMessage{
		Id:               string(message.ID),
		SentAtUnixMs:     message.SentAt.UnixMilli(),
		AuthorName:       message.AuthorName,
		AuthorAdmin:      message.AuthorAdmin,
		CountryId:        message.CountryID,
		Text:             message.Text,
		Reactions:        EncodeCounts(counts),
		ReactionsVersion: version,
	}
}

// NamedReactors is how many of the people who gave one reaction are named on the wire. A reaction everybody in
// the chat piles onto would otherwise carry a name per reader per message; the count still says how many gave
// it, so a client shows the names it has and how many more there are.
const NamedReactors = 20

func EncodeCounts(counts []reactions.Count) []*chatv1.ReactionCount {
	encoded := make([]*chatv1.ReactionCount, 0, len(counts))
	for _, count := range counts {
		encoded = append(encoded, &chatv1.ReactionCount{
			Reaction: chatv1.Reaction(count.Reaction),
			Count:    uint32(count.Count), //nolint:gosec // a count of reactors, never negative.
			Mine:     count.Mine,
			Reactors: firstNamed(count.Names),
		})
	}
	return encoded
}

func firstNamed(names []string) []string {
	if len(names) > NamedReactors {
		return names[:NamedReactors]
	}
	return names
}

// Reaction is the wire's reaction as the domain's, or reactions.ErrInvalidReaction for one the proto does not name.
func Reaction(reaction chatv1.Reaction) (reactions.Reaction, error) {
	if _, named := chatv1.Reaction_name[int32(reaction)]; !named || reaction == chatv1.Reaction_REACTION_UNSPECIFIED {
		return 0, fmt.Errorf("%w: %d", reactions.ErrInvalidReaction, reaction)
	}
	return reactions.Reaction(reaction), nil
}
