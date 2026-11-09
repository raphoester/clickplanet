package listed_battles_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers/listed_battles"
)

type query struct {
	listed []*promov1.Teaser
	err    error
	within time.Duration
}

func (q *query) Teasers(_ context.Context, within time.Duration) ([]*promov1.Teaser, error) {
	q.within = within
	return q.listed, q.err
}

func TestEachBattleListedIsTheDomainsTeaser(t *testing.T) {
	start := time.Date(2026, 11, 14, 20, 0, 0, 0, time.UTC)
	q := &query{listed: []*promov1.Teaser{{
		BattleId: "season-1/finale", StartsAt: timestamppb.New(start), EndsAt: timestamppb.New(start.Add(time.Hour)),
		Description: "The last one.", Times: 5, EndsSeason: true,
		Cards: []*promov1.TeaserCard{{BackgroundUrl: "https://media.test/bg.jpg", Headline: "Gain the most land", Line: "1st : 125 points"}},
	}}}

	battles, err := listed_battles.New(q).Teasers(t.Context(), 36*time.Hour)

	require.NoError(t, err)
	assert.Equal(t, []teasers.Teaser{{
		Battle: "season-1/finale", StartsAt: start, EndsAt: start.Add(time.Hour), Description: "The last one.", Times: 5, EndsSeason: true,
		Cards: []teasers.Card{{BackgroundURL: "https://media.test/bg.jpg", Headline: "Gain the most land", Line: "1st : 125 points"}},
	}}, battles)
	assert.Equal(t, 36*time.Hour, q.within)

	_, err = listed_battles.New(&query{err: errors.New("the gamemaster is down")}).Teasers(t.Context(), time.Hour)
	require.EqualError(t, err, "the gamemaster is down")
}
