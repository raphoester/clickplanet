package rpc_planet_play_days_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/rpc_planet_play_days"
)

const (
	ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	bob = "5a0c2f1e-7d3b-4c8a-9e6f-1b2d3c4e5f60"
)

type fakePlanet struct {
	answer *planetv1.GetPlayDaysResponse
	asked  [][]string
}

func (f *fakePlanet) GetPlayDays(
	_ context.Context,
	req *connect.Request[planetv1.GetPlayDaysRequest],
) (*connect.Response[planetv1.GetPlayDaysResponse], error) {
	f.asked = append(f.asked, req.Msg.GetAccountIds())
	return connect.NewResponse(f.answer), nil
}

func account(t *testing.T, id string) accounts.AccountID {
	t.Helper()

	account, err := accounts.AccountIDOf(id)
	require.NoError(t, err)
	return account
}

func day(t *testing.T, value string) stats.Day {
	t.Helper()

	parsed, err := stats.DayFrom(value)
	require.NoError(t, err)
	return parsed
}

func TestEachAccountsDaysAreTheOnesPlanetAnswers(t *testing.T) {
	planet := &fakePlanet{answer: &planetv1.GetPlayDaysResponse{Accounts: []*planetv1.AccountPlayDays{
		{AccountId: ada, Days: []string{"2026-09-20", "2026-10-05"}},
	}}}

	played, err := rpc_planet_play_days.New(planet).PlayDays(t.Context(), []accounts.AccountID{account(t, ada), account(t, bob)})

	require.NoError(t, err)
	assert.Equal(t, map[accounts.AccountID][]stats.Day{
		account(t, ada): {day(t, "2026-09-20"), day(t, "2026-10-05")},
	}, played, "an account with no take is left out")
	assert.Equal(t, [][]string{{ada, bob}}, planet.asked)
}

func TestNoAccountAsksNothing(t *testing.T) {
	planet := &fakePlanet{}

	played, err := rpc_planet_play_days.New(planet).PlayDays(t.Context(), nil)

	require.NoError(t, err)
	assert.Empty(t, played)
	assert.Empty(t, planet.asked)
}

func TestAnAnswerThatIsNotADayOrAnAccountIsAnError(t *testing.T) {
	for _, answered := range []*planetv1.AccountPlayDays{
		{AccountId: ada, Days: []string{"20/09/2026"}},
		{AccountId: "nope", Days: []string{"2026-09-20"}},
	} {
		planet := &fakePlanet{answer: &planetv1.GetPlayDaysResponse{Accounts: []*planetv1.AccountPlayDays{answered}}}

		_, err := rpc_planet_play_days.New(planet).PlayDays(t.Context(), []accounts.AccountID{account(t, ada)})

		require.Error(t, err, "%v", answered)
	}
}
