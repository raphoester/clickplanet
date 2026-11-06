package chatmessage

import (
	"fmt"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

func Encode(message messages.Message, counts []reactions.Count, version uint64) *chatv1.ChatMessage {
	sent := message.TakeSnapshot()

	encoded := &chatv1.ChatMessage{
		Id:               string(sent.ID),
		SentAtUnixMs:     sent.SentAt.UnixMilli(),
		AuthorAccountId:  accountOf(sent.Account),
		CountryId:        sent.Country,
		Reactions:        EncodeCounts(counts),
		ReactionsVersion: version,
		ReplyTo:          encodedQuote(sent.Quote),
	}
	if gif := EncodeGif(sent.Gif); gif != nil {
		encoded.Body = &chatv1.ChatMessage_Gif{Gif: gif}
	} else {
		encoded.Body = &chatv1.ChatMessage_Text{Text: sent.Text}
	}
	return encoded
}

func EncodeGif(gif messages.Gif) *chatv1.Gif {
	if gif.None() {
		return nil
	}
	return &chatv1.Gif{Url: gif.URL, Width: gif.Width, Height: gif.Height}
}

func Gif(gif *chatv1.Gif) messages.Gif {
	return messages.Gif{URL: gif.GetUrl(), Width: gif.GetWidth(), Height: gif.GetHeight()}
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

	encoded := &chatv1.Quote{MessageId: string(quote.MessageID), AuthorAccountId: accountOf(quote.Account)}
	if gif := EncodeGif(quote.Gif); gif != nil {
		encoded.Body = &chatv1.Quote_Gif{Gif: gif}
	} else {
		encoded.Body = &chatv1.Quote_Text{Text: quote.Text}
	}
	return encoded
}

const NamedReactors = 20

func EncodeCounts(counts []reactions.Count) []*chatv1.ReactionCount {
	encoded := make([]*chatv1.ReactionCount, 0, len(counts))
	for _, count := range counts {
		encoded = append(encoded, &chatv1.ReactionCount{
			Reaction:          chatv1.Reaction(count.Reaction()),
			Count:             uint32(count.Total()), //nolint:gosec // a count of reactors, never negative.
			Mine:              count.Mine(),
			ReactorAccountIds: AccountIDs(count.Accounts()),
		})
	}
	return encoded
}

// The first NamedReactors: a count says how many gave it, and a message everybody piles onto carries no id per reader.
func AccountIDs(accounts []messages.AccountID) []string {
	ids := make([]string, 0, min(len(accounts), NamedReactors))
	for _, account := range accounts[:min(len(accounts), NamedReactors)] {
		ids = append(ids, account.String())
	}
	return ids
}

func Reaction(reaction chatv1.Reaction) (reactions.Reaction, error) {
	if _, named := chatv1.Reaction_name[int32(reaction)]; !named || reaction == chatv1.Reaction_REACTION_UNSPECIFIED {
		return 0, fmt.Errorf("%w: %d", reactions.ErrInvalidReaction, reaction)
	}
	return reactions.Reaction(reaction), nil
}
