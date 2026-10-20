package antibot_place_shield_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase/antibot_place_shield"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

var until = time.Date(2026, 10, 21, 21, 40, 0, 0, time.UTC)

type stubUseCase struct{ ran bool }

func (s *stubUseCase) Execute(context.Context, place_shield_usecase.In) (bonuses.Held, error) {
	s.ran = true
	return bonuses.Held{Shields: 2}, nil
}

type bans map[string]time.Time

func (b bans) EnforcedSentence(_ context.Context, scope, account string) (antibot.Sentence, bool) {
	for _, key := range []string{scope, account} {
		if end, ok := b[key]; ok {
			return antibot.Sentence{Until: end}, true
		}
	}
	return antibot.Sentence{}, false
}

func requireRefusedUntil(t *testing.T, err error) {
	t.Helper()
	var refusal clicks.BanRefusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, until, refusal.Until())
}

func TestABannedCallerIsRefusedAndKeepsItsShield(t *testing.T) {
	inner := &stubUseCase{}
	decorator := antibot_place_shield.New(inner, bans{"2001:db8::/64": until})

	_, err := decorator.Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), place_shield_usecase.In{CountryID: "fr"})

	requireRefusedUntil(t, err)
	assert.False(t, inner.ran, "nothing is spent")
}

func TestABannedAccountIsRefusedFromAnyScope(t *testing.T) {
	inner := &stubUseCase{}
	decorator := antibot_place_shield.New(inner, bans{"a-guest": until})

	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), "a-guest")
	_, err := decorator.Execute(ctx, place_shield_usecase.In{CountryID: "fr"})

	requireRefusedUntil(t, err)
	assert.False(t, inner.ran)
}

func TestAnyoneElsePlacesTheShield(t *testing.T) {
	inner := &stubUseCase{}
	decorator := antibot_place_shield.New(inner, bans{"2001:db8::/64": until})

	held, err := decorator.Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), place_shield_usecase.In{CountryID: "fr"})

	require.NoError(t, err)
	assert.True(t, inner.ran)
	assert.Equal(t, 2, held.Shields)
}
