package get_history_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type stubReader struct {
	history []messages.Message
	viewers []messages.Reactor
}

func (s *stubReader) History(_ context.Context, viewer messages.Reactor) []messages.Message {
	s.viewers = append(s.viewers, viewer)
	return s.history
}

type stubAuthors struct {
	author messages.Author
	err    error
}

func (s stubAuthors) Author(context.Context, messages.AccountID, string) (messages.Author, error) {
	return s.author, s.err
}

var ada = cpsession.AccountID{15: 1}

func TestHistoryComesFromTheReader(t *testing.T) {
	reader := &stubReader{history: []messages.Message{{ID: "message-1"}, {ID: "message-2"}}}

	history := get_history_usecase.New(reader, stubAuthors{author: messages.Author{Tag: "a1b2c3"}}).
		Execute(t.Context(), cpsession.NoAccount)

	assert.Equal(t, reader.history, history)
}

func TestAPlayerReadsAsItsAccountAndAGuestAsItsTag(t *testing.T) {
	reader := &stubReader{}
	player := stubAuthors{author: messages.Author{Username: "ada", Tag: "a1b2c3"}}

	get_history_usecase.New(reader, player).Execute(t.Context(), ada)
	get_history_usecase.New(reader, player).Execute(t.Context(), cpsession.NoAccount)

	assert.Equal(t, []messages.Reactor{
		messages.ReactorOf(ada, player.author),
		messages.ReactorOf(cpsession.NoAccount, player.author),
	}, reader.viewers)
}

func TestAHistoryIsServedWhenThePlayerModuleDoesNotAnswer(t *testing.T) {
	reader := &stubReader{history: []messages.Message{{ID: "message-1"}}}

	history := get_history_usecase.New(reader, stubAuthors{err: assert.AnError}).Execute(t.Context(), ada)

	assert.Equal(t, reader.history, history)
	assert.Equal(t, []messages.Reactor{messages.NoReactor}, reader.viewers)
}
