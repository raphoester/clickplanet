package due_teasers_query_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_due_teasers_handler/due_teasers_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 13, 7, 43, 0, 0, time.UTC)

type listed struct {
	teasers []*promov1.Teaser
	within  time.Duration
}

func (l *listed) Teasers(_ context.Context, within time.Duration) ([]*promov1.Teaser, error) {
	l.within = within
	return l.teasers, nil
}

type posts struct {
	teased []teasers.BattleID
	err    error
	since  time.Time
}

func (p *posts) Teased(_ context.Context, since time.Time) ([]teasers.BattleID, error) {
	p.since = since
	return p.teased, p.err
}

func TestTheBattlesDueAreThoseWithin36HoursNoPostTeases(t *testing.T) {
	ring, finale, teased := &promov1.Teaser{BattleId: "entry-7"}, &promov1.Teaser{BattleId: "season-1/finale"}, &promov1.Teaser{BattleId: "entry-6"}
	battles := &listed{teasers: []*promov1.Teaser{teased, ring, finale}}
	read := &posts{teased: []teasers.BattleID{"entry-6", "entry-1"}}

	due, err := due_teasers_query.New(battles, read, cptime.NewFixedClock(now)).Due(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []*promov1.Teaser{ring, finale}, due)
	assert.Equal(t, 36*time.Hour, battles.within)
	assert.Equal(t, now.Add(-60*time.Hour), read.since)
}

func TestPostsThatCannotBeReadFailTheList(t *testing.T) {
	busy := &teasers.BusyError{Reason: "restarting"}

	_, err := due_teasers_query.New(&listed{}, &posts{err: busy}, cptime.NewFixedClock(now)).Due(t.Context())

	require.ErrorIs(t, err, busy)
}
