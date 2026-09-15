package get_history_usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
)

type stubReader []messages.Message

func (s stubReader) History(context.Context) []messages.Message { return s }

func TestHistoryComesFromTheReader(t *testing.T) {
	history := stubReader{{ID: "message-1"}, {ID: "message-2"}}

	assert.Equal(t, []messages.Message(history), get_history_usecase.New(history).Execute(t.Context()))
}
