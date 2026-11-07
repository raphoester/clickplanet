package audit_set_raid_mode_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/usecases/set_raid_mode_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/usecases/set_raid_mode_usecase/audit_set_raid_mode"
)

type stubExecutor struct{ err error }

func (s stubExecutor) Execute(context.Context, set_raid_mode_usecase.In) (raids.RaidMode, bool, error) {
	return raids.RaidMode{}, s.err == nil, s.err
}

var operator = moderation.OperatorOf(messages.AccountID{15: 9}, "Claude")

func audited(err error) (*audit_set_raid_mode.Audited, *bytes.Buffer) {
	var out bytes.Buffer
	return audit_set_raid_mode.New(stubExecutor{err: err}, slog.New(slog.NewTextHandler(&out, nil))), &out
}

func TestEveryRaidModeIsLoggedAtWarnWithWhoSetItAndWhy(t *testing.T) {
	executor, out := audited(nil)

	_, running, err := executor.Execute(t.Context(),
		set_raid_mode_usecase.In{Duration: time.Hour, Reason: "a raid of guests", By: operator})

	require.NoError(t, err)
	assert.True(t, running)
	assert.Contains(t, out.String(), "level=WARN")
	assert.Contains(t, out.String(), `msg="admin chat raid mode"`)
	assert.Contains(t, out.String(), "a raid of guests")
	assert.Contains(t, out.String(), operator.Account().String())
	assert.Contains(t, out.String(), "client=Claude")
}

func TestAFailureIsLoggedAndPassedOn(t *testing.T) {
	cause := errors.New("postgres is down")
	executor, out := audited(cause)

	_, _, err := executor.Execute(t.Context(), set_raid_mode_usecase.In{Duration: time.Hour, By: operator})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, out.String(), "admin chat raid mode failed")
	assert.Contains(t, out.String(), "postgres is down")
}
