package publishing_record_take_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_take_usecase/publishing_record_take"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type stubUseCase struct {
	err error
}

func (s stubUseCase) Execute(context.Context, record_take_usecase.In) error { return s.err }

var take = record_take_usecase.In{Account: players.AccountID{15: 1}, At: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)}

func TestARecordedTakeSaysTheStatsChanged(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()

	require.NoError(t, publishing_record_take.New(stubUseCase{}, events).Execute(t.Context(), take))

	require.Len(t, events.Published(), 1)
	assert.True(t, proto.Equal(&playerv1.StatsChanged{AccountId: take.Account.String()}, events.Published()[0]))
}

func TestAFailedTakePublishesNothing(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()
	failed := errors.New("postgres is down")

	err := publishing_record_take.New(stubUseCase{err: failed}, events).Execute(t.Context(), take)

	require.ErrorIs(t, err, failed)
	assert.Empty(t, events.Published())
}
