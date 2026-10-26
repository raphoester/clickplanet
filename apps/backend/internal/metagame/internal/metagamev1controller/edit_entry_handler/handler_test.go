package edit_entry_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/edit_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
)

type edit struct {
	id    schedule.EntryID
	draft schedule.Draft
}

type stubUseCase struct {
	err   error
	asked []edit
}

func (s *stubUseCase) Execute(_ context.Context, id schedule.EntryID, draft schedule.Draft) error {
	s.asked = append(s.asked, edit{id: id, draft: draft})
	return s.err
}

const first = "00000000-0000-0000-0000-000000000001"

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func TestTheEntryNamedTakesTheValuesAsked(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := edit_entry_handler.New(useCase).EditEntry(t.Context(), connect.NewRequest(&metagamev1.EditEntryRequest{
		Id: first, RuleSet: "calm", Kind: metagamev1.EntryKind_ENTRY_KIND_PLAIN, Description: "A quieter antibot.",
		StartsAt: timestamppb.New(finale), EndsAt: timestamppb.New(finale.Add(time.Hour)), Priority: 3,
	}))

	require.NoError(t, err)
	assert.Equal(t, []edit{{
		id: schedule.EntryID(uuid.UUID{15: 1}),
		draft: schedule.Draft{
			Set: "calm", Kind: schedule.Plain, Description: "A quieter antibot.", StartsAt: finale, EndsAt: finale.Add(time.Hour),
			Priority: 3,
		},
	}}, useCase.asked)
}

func TestAnEntryThatStartedIsAFailedPrecondition(t *testing.T) {
	_, err := edit_entry_handler.New(&stubUseCase{err: schedule.ErrStarted}).
		EditEntry(t.Context(), connect.NewRequest(&metagamev1.EditEntryRequest{Id: first, Kind: metagamev1.EntryKind_ENTRY_KIND_BATTLE}))

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestAnUnspecifiedKindIsAnInvalidArgumentAndAsksNothing(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := edit_entry_handler.New(useCase).EditEntry(t.Context(), connect.NewRequest(&metagamev1.EditEntryRequest{Id: first}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Empty(t, useCase.asked)
}

func TestAnIDThatIsNoEntryIsNotFoundAndAsksNothing(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := edit_entry_handler.New(useCase).EditEntry(t.Context(), connect.NewRequest(&metagamev1.EditEntryRequest{Id: "nope"}))

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	assert.Empty(t, useCase.asked)
}
