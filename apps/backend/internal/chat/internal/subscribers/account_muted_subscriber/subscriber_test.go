package account_muted_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_mute_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/account_muted_subscriber"
)

const bullyID = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

var at = time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)

type recordingUseCase struct{ ins []announce_mute_usecase.In }

func (r *recordingUseCase) Execute(_ context.Context, in announce_mute_usecase.In) error {
	r.ins = append(r.ins, in)
	return nil
}

func TestAMuteIsAnnouncedForItsAccountAtItsTimeForItsDuration(t *testing.T) {
	useCase := &recordingUseCase{}

	err := account_muted_subscriber.New(useCase).Handle(t.Context(), &chatv1.AccountMuted{
		AccountId: bullyID, MutedAt: timestamppb.New(at), Duration: durationpb.New(time.Hour),
	})

	require.NoError(t, err)
	assert.Equal(t, []announce_mute_usecase.In{
		{Account: messages.AccountIDOf(bullyID), At: at, Duration: time.Hour},
	}, useCase.ins)
}

func TestAMuteMissingAPartIsRefused(t *testing.T) {
	for name, event := range map[string]*chatv1.AccountMuted{
		"no account":  {MutedAt: timestamppb.New(at), Duration: durationpb.New(time.Hour)},
		"no time":     {AccountId: bullyID, Duration: durationpb.New(time.Hour)},
		"no duration": {AccountId: bullyID, MutedAt: timestamppb.New(at)},
	} {
		useCase := &recordingUseCase{}

		require.Error(t, account_muted_subscriber.New(useCase).Handle(t.Context(), event), name)
		assert.Empty(t, useCase.ins, name)
	}
}
