package log_set_rules_test

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo/usecases/set_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo/usecases/set_rules_usecase/log_set_rules"
)

func run(t *testing.T, in set_rules_usecase.In) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	err := log_set_rules.New(set_rules_usecase.New(tempo.NewSwitches()), slog.New(slog.NewTextHandler(&logs, nil))).
		Execute(t.Context(), in)
	return logs.String(), err
}

func TestRulesSetAreLogged(t *testing.T) {
	logs, err := run(t, set_rules_usecase.In{RefillMultiplier: 3, BoxInterval: 2 * time.Minute, GiftTag: "finale-0"})

	require.NoError(t, err)
	assert.Contains(t, logs, `level=INFO msg="rules set" refillMultiplier=3 boxInterval=2m0s gift=finale-0`)
}

func TestRulesRefusedAreLoggedAndPassedOn(t *testing.T) {
	logs, err := run(t, set_rules_usecase.In{RefillMultiplier: 0.5})

	require.ErrorIs(t, err, tempo.ErrInvalidRules)
	assert.Contains(t, logs, `level=WARN msg="refused the rules" refillMultiplier=0.5`)
}
