package planetwords_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/planetwords"
)

// Every kind a box may be is one of planet's charges, and every charge of planet's is a kind a box may be.
func TestTheOffersAndTheWalletNameTheSameKinds(t *testing.T) {
	for _, kind := range bonuses.Kinds {
		theirs := planetwords.Kind(kind)
		require.NotEqual(t, planetv1.BonusKind_BONUS_KIND_UNSPECIFIED, theirs, kind)

		ours, ok := planetwords.KindOf(theirs)
		require.True(t, ok)
		assert.Equal(t, kind, ours)
	}
	for number, name := range planetv1.BonusKind_name {
		kind := planetv1.BonusKind(number)
		if kind == planetv1.BonusKind_BONUS_KIND_UNSPECIFIED {
			continue
		}
		_, ok := planetwords.KindOf(kind)
		assert.True(t, ok, "planet holds %s, which no box may be", name)
	}
}

func TestAHandIsReadAsPlanetWroteIt(t *testing.T) {
	hand := bonuses.Hand{Refill: true, Bomb: true, Enclosures: 2, SpreadClicks: 7, Shields: 30}

	assert.Equal(t, hand, planetwords.HandOf(planetwords.Hand(hand)))
}
