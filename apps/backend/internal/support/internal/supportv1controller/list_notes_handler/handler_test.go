package list_notes_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_notes_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_notes_handler/notes_query"
)

type stubQuery struct {
	answer *supportv1.ListNotesResponse
	err    error

	since time.Time
	kind  supportv1.NoteKind
	limit uint32
}

func (s *stubQuery) Notes(
	_ context.Context, since time.Time, kind supportv1.NoteKind, limit uint32,
) (*supportv1.ListNotesResponse, error) {
	s.since, s.kind, s.limit = since, kind, limit
	return s.answer, s.err
}

func list(t *testing.T, query *stubQuery, req *supportv1.ListNotesRequest) (*supportv1.ListNotesResponse, error) {
	t.Helper()

	res, err := list_notes_handler.New(query).ListNotes(t.Context(), connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("ListNotes failed: %w", err)
	}
	return res.Msg, nil
}

func TestTheWindowReachesTheQueryAndItsAnswerComesBack(t *testing.T) {
	since := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	query := &stubQuery{answer: &supportv1.ListNotesResponse{Notes: []*supportv1.Note{{Text: "a night mode"}}}}

	answer, err := list(t, query, &supportv1.ListNotesRequest{
		Since: timestamppb.New(since), Limit: 20, Kind: supportv1.NoteKind_NOTE_KIND_BUG,
	})
	require.NoError(t, err)

	assert.True(t, since.Equal(query.since))
	assert.Equal(t, supportv1.NoteKind_NOTE_KIND_BUG, query.kind)
	assert.Equal(t, uint32(20), query.limit)
	assert.Same(t, query.answer, answer)
}

func TestNoSinceIsEveryNote(t *testing.T) {
	query := &stubQuery{answer: &supportv1.ListNotesResponse{}}

	_, err := list(t, query, &supportv1.ListNotesRequest{})
	require.NoError(t, err)

	assert.True(t, query.since.IsZero())
	assert.Equal(t, supportv1.NoteKind_NOTE_KIND_UNSPECIFIED, query.kind)
}

func TestAKindThatIsNotOneIsTheOperatorsFault(t *testing.T) {
	_, err := list(t, &stubQuery{err: fmt.Errorf("%w: 9", notes_query.ErrInvalidKind)}, &supportv1.ListNotesRequest{Kind: 9})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAFailedReadIsLeftToTheErrorNet(t *testing.T) {
	_, err := list(t, &stubQuery{err: errors.New("postgres is down")}, &supportv1.ListNotesRequest{})

	assert.ErrorContains(t, err, "postgres is down")
}
