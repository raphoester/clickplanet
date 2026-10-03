package publishing_record_message_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_message_usecase/publishing_record_message"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

type stubUseCase struct {
	err error
}

func (s stubUseCase) Execute(context.Context, record_message_usecase.In) error { return s.err }

var message = record_message_usecase.In{Account: players.AccountID{15: 1}}

func TestARecordedMessageSaysTheStatsChanged(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()

	require.NoError(t, publishing_record_message.New(stubUseCase{}, events).Execute(t.Context(), message))

	require.Len(t, events.Published(), 1)
	assert.True(t, proto.Equal(&playerv1.StatsChanged{AccountId: message.Account.String()}, events.Published()[0]))
}

func TestAFailedMessagePublishesNothing(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()
	failed := errors.New("postgres is down")

	err := publishing_record_message.New(stubUseCase{err: failed}, events).Execute(t.Context(), message)

	require.ErrorIs(t, err, failed)
	assert.Empty(t, events.Published())
}
