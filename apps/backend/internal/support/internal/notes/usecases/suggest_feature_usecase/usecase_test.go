package suggest_feature_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_note_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/usecases/suggest_feature_usecase"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now = time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC)
	ada = notes.AccountID{15: 1}
)

type failingIDs struct{}

func (failingIDs) NewID() (notes.NoteID, error) { return notes.NoteID{}, errors.New("no entropy") }

type testSuite struct {
	suite.Suite

	notes   *inmemory_note_storage.Storage
	useCase *suggest_feature_usecase.UseCase
}

func (s *testSuite) SetupTest() {
	s.notes = inmemory_note_storage.New()
	s.useCase = s.build(&notes.SequentialIDs{})
}

func (s *testSuite) build(ids notes.IDProvider) *suggest_feature_usecase.UseCase {
	return suggest_feature_usecase.New(ids, cptime.NewFixedClock(now), s.notes)
}

func (s *testSuite) suggest(useCase *suggest_feature_usecase.UseCase, text string) error {
	return useCase.Execute(s.T().Context(), suggest_feature_usecase.In{Account: ada, Text: text})
}

func (s *testSuite) TestASuggestionIsKeptCleanedWithAnIDAndTheTimeUnderItsAccount() {
	s.Require().NoError(s.suggest(s.useCase, "  a night mode\tplease  "))

	s.Equal([]notes.Snapshot{{
		ID: notes.NoteID{15: 1}, Kind: notes.KindSuggestion, Account: ada, Text: "a night mode please", At: now,
	}}, s.notes.Kept())
}

func (s *testSuite) TestASuggestionIsKeptWithWhatThePageSaid() {
	said := notes.Diagnostics{Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, GPURenderer: "Apple GPU"}

	err := s.useCase.Execute(s.T().Context(), suggest_feature_usecase.In{Account: ada, Text: "a night mode", Diagnostics: said})

	s.Require().NoError(err)
	s.Equal([]notes.Snapshot{{
		ID: notes.NoteID{15: 1}, Kind: notes.KindSuggestion, Account: ada, Text: "a night mode", Diagnostics: said, At: now,
	}}, s.notes.Kept())
}

func (s *testSuite) TestEachSuggestionGetsAnIDOfItsOwn() {
	s.Require().NoError(s.suggest(s.useCase, "one"))
	s.Require().NoError(s.suggest(s.useCase, "two"))

	kept := s.notes.Kept()
	s.Require().Len(kept, 2)
	s.NotEqual(kept[0].ID, kept[1].ID)
}

func (s *testSuite) TestAnInvalidTextIsRefusedAndNothingIsKept() {
	err := s.suggest(s.useCase, "   ")

	s.Require().ErrorIs(err, notes.ErrInvalidText)
	s.Empty(s.notes.Kept())
}

func (s *testSuite) TestASuggestionWithNoIDIsNotKept() {
	err := s.suggest(s.build(failingIDs{}), "a night mode")

	s.Require().Error(err)
	s.NotErrorIs(err, notes.ErrInvalidText)
	s.Empty(s.notes.Kept())
}

func (s *testSuite) TestASuggestionThatCannotBeKeptFails() {
	cause := errors.New("postgres is down")
	s.notes.FailWith(cause)

	s.Require().ErrorIs(s.suggest(s.useCase, "a night mode"), cause)
}
