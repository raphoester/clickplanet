package signin_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func post(clock *cptime.FixedClock) (*signin.Post, *signin.FakeMailer) {
	mailer := &signin.FakeMailer{}
	sends := cpratelimit.New("sends", cpratelimit.Config{Burst: 3, PerSecond: 1.0 / 1200}, clock)
	return signin.NewPost(signin.BlockedDomains{"mailinator.com"}, sends, mailer), mailer
}

func TestADisposableAddressGetsNothingAndSpendsNoBudget(t *testing.T) {
	p, mailer := post(cptime.NewFixedClock(now))

	for range 5 {
		require.ErrorIs(t, p.Send(t.Context(), "player@mailinator.com", signin.CodeLetter("000001")), signin.ErrAddressDisposable)
	}

	assert.Empty(t, mailer.Sent())
}

func TestAnAddressGetsThreeLettersThenOneEveryTwentyMinutes(t *testing.T) {
	clock := cptime.NewFixedClock(now)
	p, mailer := post(clock)
	for range 3 {
		require.NoError(t, p.Send(t.Context(), "player@example.com", signin.CodeLetter("000001")))
	}

	require.ErrorIs(t, p.Send(t.Context(), "player@example.com", signin.CodeLetter("000001")), signin.ErrTooManyCodes)
	require.NoError(t, p.Send(t.Context(), "other@example.com", signin.CodeLetter("000001")), "another address has its own budget")
	clock.Advance(20 * time.Minute)
	require.NoError(t, p.Send(t.Context(), "player@example.com", signin.CodeLetter("000001")))

	assert.Len(t, mailer.Sent(), 5)
}

func TestTheFormsOfOneMailboxShareItsBudgetAndEachLetterGoesWhereItWasAsked(t *testing.T) {
	p, mailer := post(cptime.NewFixedClock(now))
	for _, address := range []signin.Address{"jane+1@gmail.com", "j.a.n.e@gmail.com", "jane@googlemail.com"} {
		require.NoError(t, p.Send(t.Context(), address, signin.CodeLetter("000001")))
	}

	require.ErrorIs(t, p.Send(t.Context(), "jane+4@gmail.com", signin.CodeLetter("000001")), signin.ErrTooManyCodes,
		"a fourth form of the address is the same inbox")
	require.Len(t, mailer.Sent(), 3)
	assert.Equal(t, signin.Address("j.a.n.e@gmail.com"), mailer.Sent()[1].To)
}
