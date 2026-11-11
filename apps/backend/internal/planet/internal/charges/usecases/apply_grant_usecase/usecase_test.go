package apply_grant_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/apply_grant_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type grants struct {
	recorded *cpcolls.Set[charges.GrantID]
}

func (g grants) Record(_ context.Context, grant charges.GrantID, _ time.Time) (bool, error) {
	if g.recorded.Contains(grant) {
		return false, nil
	}
	g.recorded.Add(grant)
	return true, nil
}

type told struct {
	holder charges.Holder
	held   charges.Held
}

type recordingStreams struct {
	told []told
}

func (s *recordingStreams) PublishGranted(holder charges.Holder, held charges.Held) {
	s.told = append(s.told, told{holder: holder, held: held})
}

const ada = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var config = func() charges.Rules {
	rules := charges.TestRules()
	rules.Spread.Clicks, rules.Enclose.Held, rules.Shield.Held = 20, 3, 25
	return rules
}()

func setUp() (*apply_grant_usecase.UseCase, charges.Charges, *recordingStreams) {
	wallet := charges.NewCharges(charges.NewMemoryStore(), func() charges.Rules { return config })
	streams := &recordingStreams{}
	return apply_grant_usecase.New(cpatomic.FakeTransactioner{}, grants{recorded: cpcolls.NewSet[charges.GrantID]()}, wallet,
		streams, cptime.NewFixedClock(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC))), wallet, streams
}

func held(t *testing.T, wallet charges.Charges) charges.Held {
	t.Helper()

	holder, err := charges.ParseHolder(ada)
	require.NoError(t, err)
	held, err := wallet.Held(t.Context(), holder)
	require.NoError(t, err)
	return held
}

func TestAGrantToldTwiceIsGrantedAndToldToTheStreamsOnce(t *testing.T) {
	useCase, wallet, streams := setUp()
	in := apply_grant_usecase.In{Grant: charges.GrantID{15: 1}, Account: ada, Kind: charges.KindShields, Amount: 10}

	require.NoError(t, useCase.Execute(t.Context(), in))
	require.NoError(t, useCase.Execute(t.Context(), in))

	assert.Equal(t, 10, held(t, wallet).Shields)
	assert.Equal(t, []told{{holder: charges.Holder(ada), held: charges.Held{Shields: 10}}}, streams.told)
}

func TestTheStreamsAreToldAllTheAccountHoldsNotTheGrant(t *testing.T) {
	useCase, _, streams := setUp()

	for n, kind := range []charges.Kind{charges.KindBomb, charges.KindShields} {
		require.NoError(t, useCase.Execute(t.Context(),
			apply_grant_usecase.In{Grant: charges.GrantID{15: byte(n)}, Account: ada, Kind: kind, Amount: 10}))
	}

	require.Len(t, streams.told, 2)
	assert.Equal(t, charges.Held{Bomb: true, Shields: 10}, streams.told[1].held)
}

func TestWhatDoesNotFitTheStackIsNotGranted(t *testing.T) {
	useCase, wallet, _ := setUp()

	for n := range byte(3) {
		require.NoError(t, useCase.Execute(t.Context(),
			apply_grant_usecase.In{Grant: charges.GrantID{15: n}, Account: ada, Kind: charges.KindShields, Amount: 10}))
	}

	assert.Equal(t, config.Shield.Held, held(t, wallet).Shields)
}

func TestAGrantToNoAccountOrOfNothingIsRefusedAndToldToNobody(t *testing.T) {
	useCase, _, streams := setUp()

	for name, in := range map[string]apply_grant_usecase.In{
		"no account": {Grant: charges.GrantID{15: 1}, Account: "nope", Kind: charges.KindShields, Amount: 1},
		"nothing":    {Grant: charges.GrantID{15: 1}, Account: ada, Kind: charges.KindShields},
	} {
		require.Error(t, useCase.Execute(t.Context(), in), name)
	}
	assert.Empty(t, streams.told)
}
