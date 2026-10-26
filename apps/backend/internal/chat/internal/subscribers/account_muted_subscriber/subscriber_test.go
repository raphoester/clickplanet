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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/account_muted_subscriber"
)

const (
	bullyID = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	muteID  = "6f1c2a3b-4d5e-4f60-8a9b-0c1d2e3f4a5b"
)

var at = time.Date(2026, 10, 7, 20, 0, 0, 0, time.UTC)

type recordingUseCase struct{ ins []announce_mute_usecase.In }

func (r *recordingUseCase) Execute(_ context.Context, in announce_mute_usecase.In) error {
	r.ins = append(r.ins, in)
	return nil
}

func TestAMuteIsAnnouncedForItsAccountAtItsTimeForItsDuration(t *testing.T) {
	useCase := &recordingUseCase{}

	err := account_muted_subscriber.New(useCase).Handle(t.Context(), &chatv1.AccountMuted{
		MuteId: muteID, AccountId: bullyID, MutedAt: timestamppb.New(at), Duration: durationpb.New(time.Hour),
	})

	require.NoError(t, err)
	mute, err := mutes.MuteIDOf(muteID)
	require.NoError(t, err)
	assert.Equal(t, []announce_mute_usecase.In{
		{Mute: mute, Account: messages.AccountIDOf(bullyID), At: at, Duration: time.Hour},
	}, useCase.ins)
}

func TestAMuteMissingAPartIsRefused(t *testing.T) {
	for name, event := range map[string]*chatv1.AccountMuted{
		"no mute":     {AccountId: bullyID, MutedAt: timestamppb.New(at), Duration: durationpb.New(time.Hour)},
		"no account":  {MuteId: muteID, MutedAt: timestamppb.New(at), Duration: durationpb.New(time.Hour)},
		"no time":     {MuteId: muteID, AccountId: bullyID, Duration: durationpb.New(time.Hour)},
		"no duration": {MuteId: muteID, AccountId: bullyID, MutedAt: timestamppb.New(at)},
	} {
		useCase := &recordingUseCase{}

		require.Error(t, account_muted_subscriber.New(useCase).Handle(t.Context(), event), name)
		assert.Empty(t, useCase.ins, name)
	}
}
