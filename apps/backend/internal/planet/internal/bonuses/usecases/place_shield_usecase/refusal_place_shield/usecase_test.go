package refusal_place_shield_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase/refusal_place_shield"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/refusals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	at    = time.Date(2026, 10, 21, 20, 40, 0, 0, time.UTC)
	until = at.Add(time.Hour)
)

const account = "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"

type stubUseCase struct{ ran bool }

func (s *stubUseCase) Execute(context.Context, place_shield_usecase.In) (bonuses.Held, error) {
	s.ran = true
	return bonuses.Held{Shields: 2}, nil
}

func refusing(t *testing.T, caller refusals.Caller) refusals.Book {
	t.Helper()
	book := refusals.NewBook(refusals.NewMemoryStore(), cptime.NewFixedClock(at))
	require.NoError(t, book.Decide(t.Context(), caller, refusals.Refused(until, at)))
	return book
}

func requireRefusedUntil(t *testing.T, err error) {
	t.Helper()
	var refusal clicks.Refusal
	require.ErrorAs(t, err, &refusal)
	assert.Equal(t, until, refusal.Until())
}

func TestARefusedCallerIsRefusedAndKeepsItsShield(t *testing.T) {
	inner := &stubUseCase{}
	decorator := refusal_place_shield.New(inner, refusing(t, refusals.Caller{Scope: "2001:db8::/64"}))

	_, err := decorator.Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), place_shield_usecase.In{CountryID: "fr"})

	requireRefusedUntil(t, err)
	assert.False(t, inner.ran, "nothing is spent")
}

func TestARefusedAccountIsRefusedFromAnyScope(t *testing.T) {
	inner := &stubUseCase{}
	decorator := refusal_place_shield.New(inner, refusing(t, refusals.Caller{Account: account}))

	ctx := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), account)
	_, err := decorator.Execute(ctx, place_shield_usecase.In{CountryID: "fr"})

	requireRefusedUntil(t, err)
	assert.False(t, inner.ran)
}

func TestAnyoneElsePlacesTheShield(t *testing.T) {
	inner := &stubUseCase{}
	decorator := refusal_place_shield.New(inner, refusing(t, refusals.Caller{Scope: "2001:db8::/64"}))

	held, err := decorator.Execute(cpctx.AddIPToContext(t.Context(), "203.0.113.7"), place_shield_usecase.In{CountryID: "fr"})

	require.NoError(t, err)
	assert.True(t, inner.ran)
	assert.Equal(t, 2, held.Shields)
}
