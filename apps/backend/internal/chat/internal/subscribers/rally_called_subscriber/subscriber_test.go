package rally_called_subscriber_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/usecases/announce_rally_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/rally_called_subscriber"
)

var at = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

type recordedUseCase struct{ in []announce_rally_usecase.In }

func (r *recordedUseCase) Execute(_ context.Context, in announce_rally_usecase.In) error {
	r.in = append(r.in, in)
	return nil
}

func event() *planetv1.RallyCalled {
	return &planetv1.RallyCalled{
		RallyId: "00000000-0000-7000-8000-000000000001", AccountId: "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f", Country: "fr",
		TileId: 8, Ground: "au", LandmassId: 420, CalledAt: timestamppb.New(at),
	}
}

func TestARallyIsAnnouncedAtTheTimeItWasCalled(t *testing.T) {
	useCase := &recordedUseCase{}

	require.NoError(t, rally_called_subscriber.New(useCase).Handle(t.Context(), event()))

	assert.Equal(t, []announce_rally_usecase.In{{
		Rally: "00000000-0000-7000-8000-000000000001", Account: messages.AccountIDOf("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f"),
		Country: "fr", Ground: "au", Landmass: 420, At: at,
	}}, useCase.in)
}

func TestARallyWithNoIdNoAccountOrNoTimeIsRefused(t *testing.T) {
	for name, breaks := range map[string]func(*planetv1.RallyCalled){
		"no id":      func(e *planetv1.RallyCalled) { e.RallyId = "" },
		"no account": func(e *planetv1.RallyCalled) { e.AccountId = "" },
		"no time":    func(e *planetv1.RallyCalled) { e.CalledAt = nil },
	} {
		t.Run(name, func(t *testing.T) {
			broken := event()
			breaks(broken)
			useCase := &recordedUseCase{}

			require.Error(t, rally_called_subscriber.New(useCase).Handle(t.Context(), broken))
			assert.Empty(t, useCase.in)
		})
	}
}
