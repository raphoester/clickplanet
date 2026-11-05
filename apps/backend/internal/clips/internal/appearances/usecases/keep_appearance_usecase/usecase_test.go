package keep_appearance_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances/usecases/keep_appearance_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

type world struct {
	casts       *casts.MemoryStore
	appearances *appearances.MemoryStore
	useCase     *keep_appearance_usecase.UseCase
	ada, bob    casts.AccountID
}

func worldOf(t *testing.T) world {
	t.Helper()
	ada, err := casts.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a01")
	require.NoError(t, err)
	bob, err := casts.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a02")
	require.NoError(t, err)
	w := world{casts: casts.NewMemoryStore(), appearances: appearances.NewMemoryStore(), ada: ada, bob: bob}
	w.useCase = keep_appearance_usecase.New(w.casts, w.appearances, cptime.NewFixedClock(now))
	require.NoError(t, w.casts.Keep(t.Context(), casts.CastSnapshot{ID: casts.CastID{1}, MadeAt: now, Fighters: []casts.FighterSnapshot{
		{Number: 1, Account: bob, Name: "Bob"}, {Number: 2, Account: ada, Name: "Ada"},
	}}.Restore()))
	return w
}

func postAt(t *testing.T, id string, at time.Time) appearances.Post {
	t.Helper()
	post, err := appearances.PostOf(appearances.PostID(id), "FRANCE VS TURKEY", []appearances.Platform{appearances.TikTok}, at)
	require.NoError(t, err)
	return post
}

func TestThePlayersAPostNamesAreKept(t *testing.T) {
	w := worldOf(t)
	post := postAt(t, "p1", now.Add(5*time.Minute))

	kept, err := w.useCase.Execute(t.Context(), casts.CastID{1}, []casts.Number{2, 1}, post)
	require.NoError(t, err)

	assert.Equal(t, 2, kept)
	assert.Equal(t, []appearances.Appearance{{Post: post, Account: w.ada}, {Post: post, Account: w.bob}}, w.appearances.Kept())
}

func TestAPostOfAForgottenCastIsRefused(t *testing.T) {
	w := worldOf(t)

	_, err := w.useCase.Execute(t.Context(), casts.CastID{2}, []casts.Number{1}, postAt(t, "p1", now))

	require.ErrorIs(t, err, casts.ErrNoCast)
	assert.Empty(t, w.appearances.Kept())
}

func TestANumberTheCastDoesNotHaveKeepsNothing(t *testing.T) {
	w := worldOf(t)

	_, err := w.useCase.Execute(t.Context(), casts.CastID{1}, []casts.Number{1, 3}, postAt(t, "p1", now))

	require.ErrorIs(t, err, casts.ErrUnknownFighter)
	assert.Empty(t, w.appearances.Kept())
}

func TestAnAccountDeletedSinceTheCastIsNotNamed(t *testing.T) {
	w := worldOf(t)
	require.NoError(t, w.casts.ForgetAccount(t.Context(), w.bob))

	kept, err := w.useCase.Execute(t.Context(), casts.CastID{1}, []casts.Number{1, 2}, postAt(t, "p1", now))
	require.NoError(t, err)

	assert.Equal(t, 1, kept)
}

func TestTheAppearancesNoPlayerIsToldOfAreForgotten(t *testing.T) {
	w := worldOf(t)
	old := postAt(t, "old", now.Add(-8*24*time.Hour))
	require.NoError(t, w.appearances.Keep(t.Context(), old.AppearancesOf([]casts.AccountID{w.ada})))

	_, err := w.useCase.Execute(t.Context(), casts.CastID{1}, []casts.Number{1}, postAt(t, "p1", now))
	require.NoError(t, err)

	assert.Equal(t, []appearances.Appearance{{Post: postAt(t, "p1", now), Account: w.bob}}, w.appearances.Kept())
}
