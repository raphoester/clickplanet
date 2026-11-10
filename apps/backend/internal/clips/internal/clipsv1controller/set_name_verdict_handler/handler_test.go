package set_name_verdict_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/set_name_verdict_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts/usecases/set_verdict_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	at  = time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	ira = verdicts.AccountID{0: 0x01, 15: 2}
)

func handlerOf(t *testing.T, store *verdicts.MemoryStore) set_name_verdict_handler.SetNameVerdictHandler {
	t.Helper()
	return set_name_verdict_handler.New(set_verdict_usecase.New(store, cptime.NewFixedClock(at)))
}

func TestTheOperatorsVerdictIsAnswered(t *testing.T) {
	store := verdicts.NewMemoryStore()
	require.NoError(t, store.Keep(t.Context(), verdicts.Judged(map[verdicts.AccountID]string{ira: "IRA4Ever"},
		[]verdicts.Judgement{{Name: "IRA4Ever", Shown: true}}, at)))

	res, err := handlerOf(t, store).SetNameVerdict(t.Context(), connect.NewRequest(&clipsv1.SetNameVerdictRequest{
		AccountId: ira.String(), Shown: false, Reason: "the IRA, an armed group",
	}))

	require.NoError(t, err)
	verdict := res.Msg.GetVerdict()
	assert.Equal(t, "IRA4Ever", verdict.GetName())
	assert.False(t, verdict.GetShown())
	assert.Equal(t, "the IRA, an armed group", verdict.GetReason())
	assert.Equal(t, clipsv1.VerdictBy_VERDICT_BY_OPERATOR, verdict.GetDecidedBy())
}

func TestANameNeverJudgedIsNotFound(t *testing.T) {
	_, err := handlerOf(t, verdicts.NewMemoryStore()).SetNameVerdict(t.Context(),
		connect.NewRequest(&clipsv1.SetNameVerdictRequest{AccountId: ira.String()}))

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestABadAccountIsInvalid(t *testing.T) {
	_, err := handlerOf(t, verdicts.NewMemoryStore()).SetNameVerdict(t.Context(),
		connect.NewRequest(&clipsv1.SetNameVerdictRequest{AccountId: "IRA4Ever"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
