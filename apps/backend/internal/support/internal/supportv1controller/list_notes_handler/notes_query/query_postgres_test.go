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

var phone = notes.Diagnostics{
	Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, PixelRatio: 2.63, ConnectionDown: true,
	UserAgent: "Mozilla/5.0 (iPhone)", GPUVendor: "Apple Inc.", GPURenderer: "Apple GPU",
	WebGLVersion: "WebGL 2.0 (OpenGL ES 3.0)", MaxTextureSize: 4096, ContextLost: true, Cores: 6, MemoryGB: 0.5,
}

func (s *testSuite) suggested(n byte, account notes.AccountID, text string, at time.Time) *supportv1.Note {
	return s.suggestedFrom(n, account, text, notes.Diagnostics{}, at)
}

func (s *testSuite) suggestedFrom(n byte, account notes.AccountID, text string, said notes.Diagnostics, at time.Time) *supportv1.Note {
	note, err := notes.NewSuggestion(notes.NoteID{15: n}, account, text, said, at)
	s.Require().NoError(err)
	return s.kept(note, supportv1.NoteKind_NOTE_KIND_SUGGESTION)
}

func (s *testSuite) reported(n byte, account notes.AccountID, text string, said notes.Diagnostics, at time.Time) *supportv1.Note {
	note, err := notes.NewBugReport(notes.NoteID{15: n}, account, text, said, at)
	s.Require().NoError(err)
	return s.kept(note, supportv1.NoteKind_NOTE_KIND_BUG)
}

func (s *testSuite) kept(note notes.Note, kind supportv1.NoteKind) *supportv1.Note {
	s.Require().NoError(s.notes.Save(s.T().Context(), note))

	sent := note.TakeSnapshot()
	said := sent.Diagnostics
	return &supportv1.Note{
		Id:          uuid.UUID(sent.ID).String(),
		Kind:        kind,
		AccountId:   sent.Account.String(),
		Text:        sent.Text,
		SubmittedAt: timestamppb.New(sent.At),
		UserAgent:   said.UserAgent,
		Context: &supportv1.PageContext{
			Build: said.Build, ViewportWidth: said.ViewportWidth, ViewportHeight: said.ViewportHeight,
			ConnectionDown: said.ConnectionDown, PixelRatio: said.PixelRatio,
			GpuVendor: said.GPUVendor, GpuRenderer: said.GPURenderer, WebglVersion: said.WebGLVersion,
			MaxTextureSize: said.MaxTextureSize, ContextLost: said.ContextLost, Cores: said.Cores, MemoryGb: said.MemoryGB,
		},
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

func (s *testSuite) TestEveryNoteComesWithEverythingThePageAndTheBrowserSaid() {
	bug := s.reported(1, ada, "the globe is black", phone, start)
	bare := s.reported(2, bob, "nothing happens", notes.Diagnostics{}, start.Add(time.Minute))
	idea := s.suggestedFrom(3, bob, "a night mode", phone, start.Add(2*time.Minute))

	got := s.read(time.Time{}, 0)

	s.same([]*supportv1.Note{idea, bare, bug}, got)
	s.Equal("Apple GPU", got[0].GetContext().GetGpuRenderer())
	s.NotNil(got[1].GetContext(), "a note whose page said nothing has a context, with nothing in it")
}

func (s *testSuite) TestANoteKeptBeforeADiagnosticOrWithOneThisBuildDoesNotKnowIsStillRead() {
	_, err := s.db.ExecContext(s.T().Context(), `
		INSERT INTO notes (id, kind, account_id, text, diagnostics, submitted_at)
		VALUES ($1, 'bug', $2, 'the globe is black', '{"build": "play-a1b2c3.js", "userAgent": "Mozilla/5.0", "frameRate": 12}', $3)
	`, uuid.UUID{15: 1}, uuid.UUID(ada), start)
	s.Require().NoError(err)

	got := s.read(time.Time{}, 0)

	s.Require().Len(got, 1)
	s.Equal("play-a1b2c3.js", got[0].GetContext().GetBuild())
	s.Equal("Mozilla/5.0", got[0].GetUserAgent())
	s.Empty(got[0].GetContext().GetGpuRenderer())
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

func (s *testSuite) TestAnAnonymizedNoteIsReadWithItsTextItsDiagnosticsAndNoAccount() {
	sent := s.reported(1, ada, "the globe is black", phone, start)
	note, err := notes.NewBugReport(notes.NoteID{15: 1}, ada, "the globe is black", phone, start)
	s.Require().NoError(err)
	s.Require().NoError(s.notes.Save(s.T().Context(), note.Anonymized()))

	sent.AccountId = ""
	s.same([]*supportv1.Note{sent}, s.read(time.Time{}, 0))
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
			`INSERT INTO notes (id, kind, account_id, text, diagnostics, submitted_at) VALUES ($1, 'suggestion', $2, 'one of many', '{}', $3)`,
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
		`INSERT INTO notes (id, kind, account_id, text, diagnostics, submitted_at) VALUES ($1, 'praise', $2, 'nice game', '{}', $3)`,
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
