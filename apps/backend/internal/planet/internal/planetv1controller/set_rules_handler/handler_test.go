package set_rules_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/set_rules_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo/usecases/set_rules_usecase"
)

type stubUseCase struct {
	err  error
	seen []set_rules_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in set_rules_usecase.In) error {
	s.seen = append(s.seen, in)
	return s.err
}

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func TestTheRequestBecomesTheRules(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := set_rules_handler.New(useCase).SetRules(t.Context(), connect.NewRequest(&planetv1.SetRulesRequest{
		RefillMultiplier: 3,
		BoxIntervalMs:    120_000,
		Gift:             &planetv1.Gift{Tag: "finale-0", AccountsMadeBeforeUnixMs: finale.UnixMilli()},
		Frozen:           false,
	}))

	require.NoError(t, err)
	assert.Equal(t, []set_rules_usecase.In{{
		RefillMultiplier: 3, BoxInterval: 2 * time.Minute, GiftTag: "finale-0", GiftMadeBefore: finale,
	}}, useCase.seen)
}

func TestNoGiftAndNoCutoffAreEmpty(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := set_rules_handler.New(useCase).SetRules(t.Context(), connect.NewRequest(&planetv1.SetRulesRequest{
		Gift: &planetv1.Gift{Tag: "rehearsal"}, Frozen: true,
	}))

	require.NoError(t, err)
	assert.Equal(t, []set_rules_usecase.In{{GiftTag: "rehearsal", Frozen: true}}, useCase.seen)
}

func TestRulesNobodyCanPlayByAreAnInvalidArgument(t *testing.T) {
	_, err := set_rules_handler.New(&stubUseCase{err: tempo.ErrInvalidRules}).
		SetRules(t.Context(), connect.NewRequest(&planetv1.SetRulesRequest{RefillMultiplier: 0.5}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
