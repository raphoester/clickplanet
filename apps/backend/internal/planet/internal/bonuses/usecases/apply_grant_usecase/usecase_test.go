package apply_grant_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/apply_grant_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type grants struct {
	recorded *cpcolls.Set[bonuses.GrantID]
}

func (g grants) Record(_ context.Context, grant bonuses.GrantID, _ time.Time) (bool, error) {
	if g.recorded.Contains(grant) {
		return false, nil
	}
	g.recorded.Add(grant)
	return true, nil
}

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var config = bonuses.ChargesConfig{SpreadClicks: 20, Enclosures: 3, EnclosureMaxTiles: 50, Shields: 25}

func setUp() (*apply_grant_usecase.UseCase, bonuses.Charges) {
	charges := bonuses.NewCharges(bonuses.NewMemoryStore(), config)
	return apply_grant_usecase.New(cpatomic.FakeTransactioner{}, grants{recorded: cpcolls.NewSet[bonuses.GrantID]()}, charges,
		cptime.NewFixedClock(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC))), charges
}

func held(t *testing.T, charges bonuses.Charges) bonuses.Held {
	t.Helper()

	holder, err := bonuses.ParseHolder(ada)
	require.NoError(t, err)
	held, err := charges.Held(t.Context(), holder)
	require.NoError(t, err)
	return held
}

func TestAGrantToldTwiceIsGrantedOnce(t *testing.T) {
	useCase, charges := setUp()
	in := apply_grant_usecase.In{Grant: bonuses.GrantID{15: 1}, Account: ada, Kind: bonuses.KindShields, Amount: 10}

	require.NoError(t, useCase.Execute(t.Context(), in))
	require.NoError(t, useCase.Execute(t.Context(), in))

	assert.Equal(t, 10, held(t, charges).Shields)
}

func TestWhatDoesNotFitTheStackIsNotGranted(t *testing.T) {
	useCase, charges := setUp()

	for n := range byte(3) {
		require.NoError(t, useCase.Execute(t.Context(),
			apply_grant_usecase.In{Grant: bonuses.GrantID{15: n}, Account: ada, Kind: bonuses.KindShields, Amount: 10}))
	}

	assert.Equal(t, config.Shields, held(t, charges).Shields)
}

func TestAGrantToNoAccountOrOfNothingIsRefused(t *testing.T) {
	useCase, _ := setUp()

	for name, in := range map[string]apply_grant_usecase.In{
		"no account": {Grant: bonuses.GrantID{15: 1}, Account: "nope", Kind: bonuses.KindShields, Amount: 1},
		"nothing":    {Grant: bonuses.GrantID{15: 1}, Account: ada, Kind: bonuses.KindShields},
	} {
		assert.Error(t, useCase.Execute(t.Context(), in), name)
	}
}
