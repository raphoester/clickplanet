package chatmessage

import (
	"fmt"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

func Encode(message messages.Message, counts []reactions.Count, version uint64) *chatv1.ChatMessage {
	sent := message.TakeSnapshot()

	return &chatv1.ChatMessage{
		Id:               string(sent.ID),
		SentAtUnixMs:     sent.SentAt.UnixMilli(),
		AuthorAccountId:  accountOf(sent.Account),
		AuthorName:       sent.Author.Name(),
		AuthorAdmin:      sent.Author.Admin(),
		AuthorColor:      playerv1.NameColor(sent.Author.Color()),
		AuthorStreak:     sent.Author.Streak(),
		AuthorTitle:      encodedTitle(sent.Author.Title()),
		CountryId:        sent.Country,
		Text:             sent.Text,
		Reactions:        EncodeCounts(counts),
		ReactionsVersion: version,
		ReplyTo:          encodedQuote(sent.Quote),
	}
}

func accountOf(account messages.AccountID) string {
	if account == messages.NoAccount {
		return ""
	}
	return account.String()
}

func encodedQuote(quote messages.Quote) *chatv1.Quote {
	if quote.MessageID == "" {
		return nil
	}

	return &chatv1.Quote{
		MessageId:   string(quote.MessageID),
		AuthorName:  quote.Author.Name(),
		AuthorColor: playerv1.NameColor(quote.Author.Color()),
		Text:        quote.Text,
	}
}

func encodedTitle(title messages.Title) *playerv1.Title {
	if title.Empty() {
		return nil
	}

	encoded := &playerv1.Title{Id: title.ID(), Name: title.Name()}
	if rank := title.Rank(); !rank.Empty() {
		encoded.Rank = &playerv1.Rank{
			TrackId:   rank.TrackID(),
			TrackName: rank.TrackName(),
			Number:    rank.Number(),
			Count:     rank.Count(),
		}
	}
	return encoded
}

const NamedReactors = 20

func EncodeCounts(counts []reactions.Count) []*chatv1.ReactionCount {
	encoded := make([]*chatv1.ReactionCount, 0, len(counts))
	for _, count := range counts {
		encoded = append(encoded, &chatv1.ReactionCount{
			Reaction: chatv1.Reaction(count.Reaction()),
			Count:    uint32(count.Total()), //nolint:gosec // a count of reactors, never negative.
			Mine:     count.Mine(),
			Reactors: firstNamed(count.Names()),
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
