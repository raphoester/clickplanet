package reassign_country_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/reassign_country_handler"
)

type stubUseCase struct {
	in  reassign_country_usecase.In
	out reassign_country_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in reassign_country_usecase.In) (reassign_country_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func reassign(t *testing.T, useCase *stubUseCase) (*planetv1.ReassignCountryResponse, error) {
	t.Helper()

	res, err := reassign_country_handler.New(useCase).ReassignCountry(t.Context(),
		connect.NewRequest(&planetv1.ReassignCountryRequest{FromCountryId: "dz", ToCountryId: "fr", DryRun: true}))
	if err != nil {
		return nil, fmt.Errorf("reassignment refused: %w", err)
	}

	return res.Msg, nil
}

func TestTheRequestReachesTheUseCaseAndTheCountsComeBack(t *testing.T) {
	useCase := &stubUseCase{out: reassign_country_usecase.Out{FromBefore: 22040, ToBefore: 3, Moved: 22040, FromAfter: 0, ToAfter: 22043}}

	res, err := reassign(t, useCase)
	require.NoError(t, err)

	assert.Equal(t, reassign_country_usecase.In{From: "dz", To: "fr", DryRun: true}, useCase.in)
	assert.Equal(t, uint32(22040), res.GetFromBefore())
	assert.Equal(t, uint32(3), res.GetToBefore())
	assert.Equal(t, uint32(22040), res.GetMoved())
	assert.Equal(t, uint32(0), res.GetFromAfter())
	assert.Equal(t, uint32(22043), res.GetToAfter())
}

func TestACallerMistakeIsInvalidArgument(t *testing.T) {
	for _, err := range []error{
		fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, "xx"),
		fmt.Errorf("%w: %q", reassign_country_usecase.ErrSameCountry, "fr"),
	} {
		_, got := reassign(t, &stubUseCase{err: err})
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(got), err.Error())
	}
}

func TestAnythingElseIsLeftToTheErrorNet(t *testing.T) {
	cause := errors.New("reassignment interrupted: context canceled")

	_, err := reassign(t, &stubUseCase{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err), "an unrecognised error must reach the net unwrapped")
}
