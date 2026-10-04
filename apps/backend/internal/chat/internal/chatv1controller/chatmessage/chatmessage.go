package chatmessage

import (
	"fmt"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

func Encode(message messages.Message, counts []reactions.Count, version uint64) *chatv1.ChatMessage {
	return &chatv1.ChatMessage{
		Id:               string(message.ID),
		SentAtUnixMs:     message.SentAt.UnixMilli(),
		AuthorName:       message.AuthorName,
		AuthorAdmin:      message.AuthorAdmin,
		AuthorColor:      playerv1.NameColor(message.AuthorColor),
		AuthorStreak:     message.AuthorStreak,
		AuthorTitle:      encodedTitle(message.AuthorTitle),
		CountryId:        message.CountryID,
		Text:             message.Text,
		Reactions:        EncodeCounts(counts),
		ReactionsVersion: version,
	}
}

func encodedTitle(title messages.Title) *playerv1.Title {
	if title.Empty() {
		return nil
	}

	encoded := &playerv1.Title{Id: title.ID, Name: title.Name}
	if !title.Rank.Empty() {
		encoded.Rank = &playerv1.Rank{
			TrackId:   title.Rank.TrackID,
			TrackName: title.Rank.TrackName,
			Number:    title.Rank.Number,
			Count:     title.Rank.Count,
		}
	}
	return encoded
}

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

func Reaction(reaction chatv1.Reaction) (reactions.Reaction, error) {
	if _, named := chatv1.Reaction_name[int32(reaction)]; !named || reaction == chatv1.Reaction_REACTION_UNSPECIFIED {
		return 0, fmt.Errorf("%w: %d", reactions.ErrInvalidReaction, reaction)
	}
	return reactions.Reaction(reaction), nil
}
