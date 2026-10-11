package notes_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/postgres_note_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_notes_handler/notes_query"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	notes *postgres_note_store.Store
	query *notes_query.PostgresQuery
}

var (
	ada   = notes.AccountID{15: 1}
	bob   = notes.AccountID{15: 2}
	start = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "support", migrations.FS)
	s.notes = postgres_note_store.New(s.db)
	s.query = notes_query.NewPostgresQuery(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) suggested(n byte, account notes.AccountID, text string, at time.Time) *supportv1.Note {
	id := notes.NoteID{15: n}
	note, err := notes.NewSuggestion(id, account, text, at)
	s.Require().NoError(err)
	s.Require().NoError(s.notes.Save(s.T().Context(), note))

	return &supportv1.Note{
		Id:          uuid.UUID(id).String(),
		Kind:        supportv1.NoteKind_NOTE_KIND_SUGGESTION,
		AccountId:   account.String(),
		Text:        text,
		SubmittedAt: timestamppb.New(at),
	}
}

func (s *testSuite) reported(n byte, account notes.AccountID, text string, diagnostics notes.Diagnostics, at time.Time) *supportv1.Note {
	id := notes.NoteID{15: n}
	note, err := notes.NewBugReport(id, account, text, diagnostics, at)
	s.Require().NoError(err)
	s.Require().NoError(s.notes.Save(s.T().Context(), note))

	return &supportv1.Note{
		Id:          uuid.UUID(id).String(),
		Kind:        supportv1.NoteKind_NOTE_KIND_BUG,
		AccountId:   account.String(),
		Text:        text,
		SubmittedAt: timestamppb.New(at),
	}
}

func (s *testSuite) read(since time.Time, limit uint32) []*supportv1.Note {
	return s.readKind(since, supportv1.NoteKind_NOTE_KIND_UNSPECIFIED, limit)
}

func (s *testSuite) readKind(since time.Time, kind supportv1.NoteKind, limit uint32) []*supportv1.Note {
	answer, err := s.query.Notes(s.T().Context(), since, kind, limit)
	s.Require().NoError(err)
	return answer.GetNotes()
}

func (s *testSuite) same(want []*supportv1.Note, got []*supportv1.Note) {
	s.True(proto.Equal(&supportv1.ListNotesResponse{Notes: want}, &supportv1.ListNotesResponse{Notes: got}),
		"want %v, got %v", want, got)
}

func (s *testSuite) TestNoNoteIsAnEmptyAnswer() {
	s.Empty(s.read(time.Time{}, 0))
}

func (s *testSuite) TestEveryNoteComesNewestFirstWithItsAccount() {
	first := s.suggested(1, ada, "a night mode", start)
	second := s.suggested(2, bob, "more flags\nand a bigger chat", start.Add(time.Minute))

	s.same([]*supportv1.Note{second, first}, s.read(time.Time{}, 0))
}

func (s *testSuite) TestABugReportComesWithWhatThePageAndTheBrowserSaid() {
	phone := s.reported(1, ada, "the globe is black",
		notes.Diagnostics{
			Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, ConnectionDown: true, UserAgent: "Mozilla/5.0 (iPhone)",
		}, start)
	phone.Context = &supportv1.BugContext{Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, ConnectionDown: true}
	phone.UserAgent = "Mozilla/5.0 (iPhone)"
	bare := s.reported(2, bob, "nothing happens", notes.Diagnostics{}, start.Add(time.Minute))
	idea := s.suggested(3, bob, "a night mode", start.Add(2*time.Minute))

	s.same([]*supportv1.Note{idea, bare, phone}, s.read(time.Time{}, 0))
}

func (s *testSuite) TestAKindAskedForLeavesTheOthersOut() {
	bug := s.reported(1, ada, "the globe is black", notes.Diagnostics{}, start)
	idea := s.suggested(2, bob, "a night mode", start.Add(time.Minute))

	s.same([]*supportv1.Note{bug}, s.readKind(time.Time{}, supportv1.NoteKind_NOTE_KIND_BUG, 0))
	s.same([]*supportv1.Note{idea}, s.readKind(time.Time{}, supportv1.NoteKind_NOTE_KIND_SUGGESTION, 0))
}

func (s *testSuite) TestAKindThatIsNotOneIsRefused() {
	_, err := s.query.Notes(s.T().Context(), time.Time{}, supportv1.NoteKind(99), 0)

	s.Require().ErrorIs(err, notes_query.ErrInvalidKind)
}

func (s *testSuite) TestEveryKindTheStoreKeepsIsRead() {
	s.suggested(1, ada, "a night mode", start)
	s.reported(2, ada, "the globe is black", notes.Diagnostics{}, start.Add(time.Minute))

	got := s.read(time.Time{}, 0)

	s.Require().Len(got, 2)
	s.Equal(supportv1.NoteKind_NOTE_KIND_BUG, got[0].GetKind())
	s.Equal(supportv1.NoteKind_NOTE_KIND_SUGGESTION, got[1].GetKind())
}

func (s *testSuite) TestNotesBeforeSinceAreLeftOut() {
	s.suggested(1, ada, "old", start)
	atSince := s.suggested(2, ada, "at since", start.Add(time.Hour))
	later := s.suggested(3, bob, "later", start.Add(2*time.Hour))

	s.same([]*supportv1.Note{later, atSince}, s.read(start.Add(time.Hour), 0))
}

func (s *testSuite) TestTheLimitKeepsTheNewest() {
	s.suggested(1, ada, "old", start)
	newest := s.suggested(2, ada, "newest", start.Add(time.Minute))

	s.same([]*supportv1.Note{newest}, s.read(time.Time{}, 1))
}

func (s *testSuite) TestNoLimitIsTheDefaultAndNoLimitPassesTheMost() {
	for n := range notes_query.MaxLimit + 5 {
		_, err := s.db.ExecContext(s.T().Context(),
			`INSERT INTO notes (id, kind, account_id, text, submitted_at) VALUES ($1, 'suggestion', $2, 'one of many', $3)`,
			uuid.New(), uuid.UUID(ada), start.Add(time.Duration(n)*time.Second))
		s.Require().NoError(err)
	}

	s.Len(s.read(time.Time{}, 0), notes_query.DefaultLimit)
	s.Len(s.read(time.Time{}, notes_query.MaxLimit+5), notes_query.MaxLimit)
}

func (s *testSuite) TestAKindTheDatabaseNamesAndTheWireCannotCarryRefusesTheRead() {
	_, err := s.db.ExecContext(s.T().Context(), `ALTER TYPE note_kind ADD VALUE IF NOT EXISTS 'praise'`)
	s.Require().NoError(err)
	_, err = s.db.ExecContext(s.T().Context(),
		`INSERT INTO notes (id, kind, account_id, text, submitted_at) VALUES ($1, 'praise', $2, 'nice game', $3)`,
		uuid.New(), uuid.UUID(ada), start)
	s.Require().NoError(err)

	_, err = s.query.Notes(s.T().Context(), time.Time{}, supportv1.NoteKind_NOTE_KIND_UNSPECIFIED, 0)

	s.Require().ErrorIs(err, notes_query.ErrUnknownKind)
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := s.query.Notes(ctx, time.Time{}, supportv1.NoteKind_NOTE_KIND_UNSPECIFIED, 0)

	s.Error(err, "not an empty list")
}
