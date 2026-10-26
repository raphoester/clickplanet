package add_entry_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	rulesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/rules/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/rulesv1controller/add_entry_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule"
)

type stubUseCase struct {
	err   error
	asked []schedule.Draft
}

func (s *stubUseCase) Execute(_ context.Context, draft schedule.Draft) (schedule.EntryID, error) {
	s.asked = append(s.asked, draft)
	return schedule.EntryID(uuid.UUID{15: 1}), s.err
}

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func add(t *testing.T, useCase *stubUseCase, req *rulesv1.AddEntryRequest) (*connect.Response[rulesv1.AddEntryResponse], error) {
	t.Helper()
	return add_entry_handler.New(useCase).AddEntry(t.Context(), connect.NewRequest(req)) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestTheEntryAskedIsAddedAndNamed(t *testing.T) {
	useCase := &stubUseCase{}

	res, err := add(t, useCase, &rulesv1.AddEntryRequest{
		RuleSet: "rush", StartsAt: timestamppb.New(finale), EndsAt: timestamppb.New(finale.Add(time.Hour)), Priority: 2,
	})

	require.NoError(t, err)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", res.Msg.GetId())
	require.Len(t, useCase.asked, 1)
	assert.Equal(t, schedule.Draft{Set: "rush", StartsAt: finale, EndsAt: finale.Add(time.Hour), Priority: 2}, useCase.asked[0])
}

func TestAnUnsetTimeIsNone(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := add(t, useCase, &rulesv1.AddEntryRequest{RuleSet: "rush", Priority: 1})

	require.NoError(t, err)
	assert.True(t, useCase.asked[0].StartsAt.IsZero())
	assert.True(t, useCase.asked[0].EndsAt.IsZero())
}

func TestEachRefusalHasItsCode(t *testing.T) {
	for err, code := range map[error]connect.Code{
		schedule.ErrInvalidEntry: connect.CodeInvalidArgument,
		schedule.ErrUnknownSet:   connect.CodeFailedPrecondition,
	} {
		_, got := add(t, &stubUseCase{err: err}, &rulesv1.AddEntryRequest{})

		assert.Equal(t, code, connect.CodeOf(got), "%v", err)
	}
}

func TestAFailureIsTheErrorNets(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := add(t, &stubUseCase{err: down}, &rulesv1.AddEntryRequest{})

	require.ErrorIs(t, err, down)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
