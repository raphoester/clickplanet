package delete_entry_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/delete_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type stubUseCase struct {
	err   error
	asked []schedule.EntryID
}

func (s *stubUseCase) Execute(_ context.Context, id schedule.EntryID) error {
	s.asked = append(s.asked, id)
	return s.err
}

const first = "00000000-0000-0000-0000-000000000001"

func TestTheEntryNamedIsDeleted(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := delete_entry_handler.New(useCase).DeleteEntry(t.Context(), connect.NewRequest(&gamemasterv1.DeleteEntryRequest{Id: first}))

	require.NoError(t, err)
	assert.Equal(t, []schedule.EntryID{schedule.EntryID(uuid.UUID{15: 1})}, useCase.asked)
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for err, code := range map[error]connect.Code{
		schedule.ErrStarted:      connect.CodeFailedPrecondition,
		schedule.ErrUnknownEntry: connect.CodeNotFound,
	} {
		_, got := delete_entry_handler.New(&stubUseCase{err: err}).
			DeleteEntry(t.Context(), connect.NewRequest(&gamemasterv1.DeleteEntryRequest{Id: first}))

		assert.Equal(t, code, connect.CodeOf(got), "%v", err)
	}
}
