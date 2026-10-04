package publishing_count_takes_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/usecases/count_takes_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/usecases/count_takes_usecase/publishing_count_takes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type stubExecutor struct {
	out count_takes_usecase.Out
	err error
}

func (s stubExecutor) Execute(context.Context) (count_takes_usecase.Out, error) {
	return s.out, s.err
}

func TestEachAccountCountedIsToldItsStatsChangedOnce(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()
	ada, bob := players.AccountID{15: 1}, players.AccountID{15: 2}

	_, err := publishing_count_takes.New(stubExecutor{out: count_takes_usecase.Out{Takes: 5, Counted: []players.AccountID{ada, bob}}}, events).
		Execute(t.Context())

	require.NoError(t, err)
	published := events.Published()
	require.Len(t, published, 2)
	assert.True(t, proto.Equal(&playerv1.StatsChanged{AccountId: ada.String()}, published[0]))
	assert.True(t, proto.Equal(&playerv1.StatsChanged{AccountId: bob.String()}, published[1]))
}

func TestAFailedCountTellsNothing(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()
	cause := errors.New("postgres is down")

	_, err := publishing_count_takes.New(stubExecutor{err: cause}, events).Execute(t.Context())

	require.ErrorIs(t, err, cause)
	assert.Empty(t, events.Published())
}
