package list_due_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/usecases/list_due_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 13, 7, 43, 0, 0, time.UTC)

type fakeBackend struct {
	listed []teasers.Teaser
	teased []teasers.BattleID
	err    error
	within time.Duration
	since  time.Time
}

func (f *fakeBackend) Teasers(_ context.Context, within time.Duration) ([]teasers.Teaser, error) {
	f.within = within
	return f.listed, f.err
}

func (f *fakeBackend) Teased(_ context.Context, since time.Time) ([]teasers.BattleID, error) {
	f.since = since
	return f.teased, nil
}

type fakeStore struct{ kept []teasers.Teaser }

func (f *fakeStore) Keep(_ context.Context, teaser teasers.Teaser) (string, error) {
	f.kept = append(f.kept, teaser)
	return teaser.FolderName(), nil
}

func TestEachBattleNoPostTeasesIsKeptForTheRenderer(t *testing.T) {
	ring := teasers.Teaser{Battle: "entry-7", StartsAt: now.Add(32 * time.Hour)}
	finale := teasers.Teaser{Battle: "season-1/finale", StartsAt: now.Add(35 * time.Hour), EndsSeason: true}
	teased := teasers.Teaser{Battle: "entry-6", StartsAt: now.Add(5 * time.Hour)}
	backend := &fakeBackend{listed: []teasers.Teaser{teased, ring, finale}, teased: []teasers.BattleID{"entry-6"}}
	store := &fakeStore{}

	due, err := list_due_usecase.New(backend, backend, store, cptime.NewFixedClock(now)).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []list_due_usecase.Due{{Folder: "entry-7", Teaser: ring}, {Folder: "season-1_finale", Teaser: finale}}, due)
	assert.Equal(t, []teasers.Teaser{ring, finale}, store.kept)
	assert.Equal(t, teasers.Ahead, backend.within)
	assert.Equal(t, teasers.TeasedSince(now), backend.since)
}

func TestNoBattleToComeIsNothingDue(t *testing.T) {
	due, err := list_due_usecase.New(&fakeBackend{}, &fakeBackend{}, &fakeStore{}, cptime.NewFixedClock(now)).Execute(t.Context())

	require.NoError(t, err)
	assert.Empty(t, due)
}

func TestABattleListPromoDoesNotGiveFailsTheRun(t *testing.T) {
	backend := &fakeBackend{err: errors.New("ssh failed")}

	_, err := list_due_usecase.New(backend, backend, &fakeStore{}, cptime.NewFixedClock(now)).Execute(t.Context())

	require.ErrorContains(t, err, "failed to list the battles to come: ssh failed")
}
