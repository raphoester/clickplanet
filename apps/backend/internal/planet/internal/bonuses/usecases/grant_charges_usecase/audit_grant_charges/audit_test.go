package audit_grant_charges_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/grant_charges_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/grant_charges_usecase/audit_grant_charges"
)

type stubUseCase struct {
	out grant_charges_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context, grant_charges_usecase.In) (grant_charges_usecase.Out, error) {
	return s.out, s.err
}

func TestAGrantIsLoggedWithWhatTheAccountHolds(t *testing.T) {
	var logs bytes.Buffer
	want := grant_charges_usecase.Out{Holder: "acc-1", After: bonuses.Held{Bomb: true, SpreadClicks: 4}}
	useCase := audit_grant_charges.New(stubUseCase{out: want}, slog.New(slog.NewTextHandler(&logs, nil)))

	out, err := useCase.Execute(t.Context(), grant_charges_usecase.In{
		Account: "acc-1",
		Grant:   bonuses.Held{Bomb: true, SpreadClicks: 4},
	})
	require.NoError(t, err)

	assert.Equal(t, want, out)
	assert.Contains(t, logs.String(),
		`level=WARN msg="admin charge grant" asked=acc-1 refill=false bomb=true enclosures=0 spreadClicks=4 defenders=0 account=acc-1`)
	assert.Contains(t, logs.String(), "after=\"{Refill:false Bomb:true Enclosures:0 SpreadClicks:4 Defenders:0}\"")
}

func TestARefusedGrantIsLoggedToo(t *testing.T) {
	var logs bytes.Buffer
	cause := errors.New("not an account id")
	useCase := audit_grant_charges.New(stubUseCase{err: cause}, slog.New(slog.NewTextHandler(&logs, nil)))

	_, err := useCase.Execute(t.Context(), grant_charges_usecase.In{Account: "bot"})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs.String(), `msg="admin charge grant failed" asked=bot`)
}
