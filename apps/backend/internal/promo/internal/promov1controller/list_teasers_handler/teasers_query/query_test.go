package teasers_query_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/get_teaser_handler/teaser_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/promov1controller/list_teasers_handler/teasers_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)

type timeline struct {
	battles []*gamemasterv1.Battle
	err     error
}

func (t timeline) Battles(context.Context) ([]*gamemasterv1.Battle, error) {
	return t.battles, t.err
}

// Each battle's cards by its id, and the ids asked.
type cards struct {
	by    map[string][]*promov1.TeaserCard
	err   map[string]error
	asked []string
}

func (c *cards) Teaser(_ context.Context, battleID string) (*promov1.GetTeaserResponse, error) {
	c.asked = append(c.asked, battleID)
	if err := c.err[battleID]; err != nil {
		return nil, err
	}
	return &promov1.GetTeaserResponse{Cards: c.by[battleID]}, nil
}

func battleAt(id string, in time.Duration) *gamemasterv1.Battle {
	return &gamemasterv1.Battle{
		EntryId: id, StartsAt: timestamppb.New(now.Add(in)), EndsAt: timestamppb.New(now.Add(in + time.Hour)),
		Description: "Ring War: close a ring around any land.", Times: 2,
	}
}

var card = &promov1.TeaserCard{BackgroundUrl: "https://media.clickplanet.lol/promo/backgrounds/a", Headline: "The inside is yours"}

func TestTheBattlesOfTheWindowAreListedByStartWithTheirCards(t *testing.T) {
	finale := &gamemasterv1.Battle{
		RoundId: "season-1/finale", StartsAt: timestamppb.New(now.Add(30 * time.Hour)), EndsAt: timestamppb.New(now.Add(34 * time.Hour)),
		Times: 3, EndsSeason: true,
	}
	battles := timeline{battles: []*gamemasterv1.Battle{battleAt("ring-war", 12*time.Hour), finale, battleAt("next-week", 6*24*time.Hour)}}
	sets := &cards{by: map[string][]*promov1.TeaserCard{"ring-war": {card}, "season-1/finale": {card, card}, "next-week": {card}}}

	teasers, err := teasers_query.New(battles, sets, cptime.NewFixedClock(now)).Teasers(t.Context(), 36*time.Hour)

	require.NoError(t, err)
	require.Len(t, teasers, 2)
	assert.Equal(t, "ring-war", teasers[0].GetBattleId())
	assert.Equal(t, "Ring War: close a ring around any land.", teasers[0].GetDescription())
	assert.Equal(t, now.Add(12*time.Hour), teasers[0].GetStartsAt().AsTime())
	assert.Equal(t, uint32(2), teasers[0].GetTimes())
	assert.Len(t, teasers[0].GetCards(), 1)
	assert.Equal(t, "season-1/finale", teasers[1].GetBattleId(), "a finale by its round")
	assert.True(t, teasers[1].GetEndsSeason())
	assert.Equal(t, []string{"ring-war", "season-1/finale"}, sets.asked, "a battle past the window is not read")
}

func TestABattleWithNoCardOrThatTheScheduleDroppedIsLeftOut(t *testing.T) {
	battles := timeline{battles: []*gamemasterv1.Battle{battleAt("plain", time.Hour), battleAt("dropped", 2*time.Hour), battleAt("teased", 3*time.Hour)}}
	sets := &cards{
		by:  map[string][]*promov1.TeaserCard{"teased": {card}},
		err: map[string]error{"dropped": fmt.Errorf("failed: %w", teaser_query.ErrNoBattle)},
	}

	teasers, err := teasers_query.New(battles, sets, cptime.NewFixedClock(now)).Teasers(t.Context(), 24*time.Hour)

	require.NoError(t, err)
	require.Len(t, teasers, 1)
	assert.Equal(t, "teased", teasers[0].GetBattleId())
}

func TestCardsOrATimelineThatCannotBeReadFailTheRead(t *testing.T) {
	broken := errors.New("planet does not answer")
	sets := &cards{err: map[string]error{"ring-war": broken}}

	_, err := teasers_query.New(timeline{battles: []*gamemasterv1.Battle{battleAt("ring-war", time.Hour)}}, sets, cptime.NewFixedClock(now)).
		Teasers(t.Context(), 24*time.Hour)
	require.ErrorIs(t, err, broken)

	_, err = teasers_query.New(timeline{err: broken}, sets, cptime.NewFixedClock(now)).Teasers(t.Context(), 24*time.Hour)
	require.ErrorIs(t, err, broken)
}
