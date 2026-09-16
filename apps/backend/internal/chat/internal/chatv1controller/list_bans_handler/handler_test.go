package list_bans_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/list_bans_handler"
)

type stubUseCase []bans.Ban

func (s stubUseCase) Execute(context.Context) []bans.Ban { return s }

var noon = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

func listBans(t *testing.T, useCase stubUseCase) *chatv1.ListBansResponse {
	t.Helper()

	res, err := list_bans_handler.New(useCase).
		ListBans(t.Context(), connect.NewRequest(&chatv1.ListBansRequest{}))
	require.NoError(t, err)

	return res.Msg
}

func TestListBansMapsEveryBanInOrder(t *testing.T) {
	msg := listBans(t, stubUseCase{
		{AuthorTag: "a1b2c3", BannedAt: noon, Reason: "slurs"},
		{AuthorTag: "d4e5f6", BannedAt: noon.Add(-time.Hour), Reason: "spam"},
	})

	require.Len(t, msg.GetBans(), 2)
	assert.Equal(t, "a1b2c3", msg.GetBans()[0].GetAuthorTag())
	assert.Equal(t, "slurs", msg.GetBans()[0].GetReason())
	assert.Equal(t, noon, msg.GetBans()[0].GetBannedAt().AsTime())
	assert.Equal(t, "d4e5f6", msg.GetBans()[1].GetAuthorTag())
}

func TestListBansAnswersAnEmptyListRatherThanNothing(t *testing.T) {
	assert.Empty(t, listBans(t, stubUseCase{}).GetBans())
}
