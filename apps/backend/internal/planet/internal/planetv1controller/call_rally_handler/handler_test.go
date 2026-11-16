package call_rally_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/call_rally_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies/usecases/call_rally_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var calledAt = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

type stubUseCase struct {
	rally rallies.Rally
	err   error
	in    call_rally_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in call_rally_usecase.In) (rallies.Rally, error) {
	s.in = in
	return s.rally, s.err
}

func call(t *testing.T, useCase call_rally_handler.UseCase) (*planetv1.CallRallyResponse, error) {
	t.Helper()

	res, err := call_rally_handler.New(useCase).CallRally(t.Context(), connect.NewRequest(&planetv1.CallRallyRequest{
		Point: &planetv1.GlobePoint{X: 0.6, Z: 0.8}, CountryId: "fr",
	}))
	if err != nil {
		return nil, fmt.Errorf("call refused: %w", err)
	}
	return res.Msg, nil
}

func detailOf(t *testing.T, err error) proto.Message {
	t.Helper()
	var refused *connect.Error
	require.ErrorAs(t, err, &refused)
	require.Len(t, refused.Details(), 1)
	detail, detailErr := refused.Details()[0].Value()
	require.NoError(t, detailErr)
	return detail
}

func TestARallyCalledIsAnsweredWithAllItsTimeLeft(t *testing.T) {
	useCase := &stubUseCase{rally: rallies.Snapshot{
		ID: rallies.RallyID(uuid.MustParse("00000000-0000-7000-8000-000000000001")), Flag: "fr", Centre: clicks.Vec3{X: 0.6, Z: 0.8}, Radius: 0.072,
		Account:  cpsession.AccountID(uuid.MustParse("01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f")),
		CalledAt: calledAt, EndsAt: calledAt.Add(3 * time.Minute),
	}.Restore()}

	answer, err := call(t, useCase)

	require.NoError(t, err)
	assert.Equal(t, call_rally_usecase.In{Point: clicks.Vec3{X: 0.6, Z: 0.8}, CountryID: "fr"}, useCase.in)
	assert.True(t, proto.Equal(&planetv1.Rally{
		RallyId: "00000000-0000-7000-8000-000000000001", AccountId: "01926c6e-7a4b-7c3d-8e9f-0a1b2c3d4e5f", CountryId: "fr",
		Point: &planetv1.GlobePoint{X: 0.6, Z: 0.8}, Radius: 0.072, LeftMs: 180_000,
	}, answer.GetRally()), "%v", answer)
}

func TestARallyStillOnIsAFailedPreconditionWithARallyRefusal(t *testing.T) {
	_, err := call(t, &stubUseCase{err: fmt.Errorf("account: %w", rallies.ErrRallyOn)})

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.IsType(t, &planetv1.RallyRefusal{}, detailOf(t, err))
}

func TestCallingNoRallyIsNotFound(t *testing.T) {
	_, err := call(t, &stubUseCase{err: call_rally_usecase.ErrNoRally})

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestABannedCallerIsDeniedAndToldUntilWhen(t *testing.T) {
	_, err := call(t, &stubUseCase{err: clicks.NewRefusal(calledAt.Add(time.Hour))})

	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	assert.IsType(t, &planetv1.BanRefusal{}, detailOf(t, err))
}

func TestARallyForAnotherFlagIsRefusedWithTheFlagHeld(t *testing.T) {
	_, err := call(t, &stubUseCase{err: allegiances.Refusal{Flag: "de", Reason: allegiances.ErrOtherFlag}})

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	assert.IsType(t, &planetv1.Allegiance{}, detailOf(t, err))
}

func TestAMalformedCallIsAnInvalidArgument(t *testing.T) {
	for _, cause := range []error{clicks.ErrUnknownCountry, rallies.ErrNoPoint} {
		_, err := call(t, &stubUseCase{err: fmt.Errorf("%w: detail", cause)})
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
}
