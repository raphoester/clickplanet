package get_history_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
)

type stubReader []messages.Message

func (s stubReader) History(context.Context) []messages.Message {
	return append(make([]messages.Message, 0, len(s)), s...)
}

type stubBans []string

func (s stubBans) Banned(tag string) bool {
	for _, banned := range s {
		if banned == tag {
			return true
		}
	}
	return false
}

func TestHistoryComesFromTheReader(t *testing.T) {
	history := stubReader{{ID: "message-1"}, {ID: "message-2"}}

	read := get_history_usecase.New(history, stubBans{}).Execute(t.Context())

	assert.Equal(t, []messages.Message(history), read)
}

func TestABannedAuthorKeepsEverythingButTheText(t *testing.T) {
	history := stubReader{
		{ID: "message-1", AuthorName: "Ana", AuthorTag: "a1b2c3", CountryID: "fr", Text: "hello"},
		{ID: "message-2", AuthorName: "Bo", AuthorTag: "d4e5f6", CountryID: "de", Text: "hi"},
	}

	read := get_history_usecase.New(history, stubBans{"a1b2c3"}).Execute(t.Context())

	assert.Equal(t, messages.Message{
		ID: "message-1", AuthorName: "Ana", AuthorTag: "a1b2c3", CountryID: "fr", Redacted: true,
	}, read[0])
	assert.Equal(t, history[1], read[1])
}

func TestLiftingABanNeedsNothingPutBack(t *testing.T) {
	history := stubReader{{ID: "message-1", AuthorTag: "a1b2c3", Text: "hello"}}

	get_history_usecase.New(history, stubBans{"a1b2c3"}).Execute(t.Context())
	read := get_history_usecase.New(history, stubBans{}).Execute(t.Context())

	assert.Equal(t, "hello", read[0].Text)
}
