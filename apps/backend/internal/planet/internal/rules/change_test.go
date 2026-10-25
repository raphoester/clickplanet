package rules_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func TestANewRequestIsAChangeToItsSet(t *testing.T) {
	change, err := rules.NewChange("season-0:finale", rules.Set{Name: "rush"}, rules.SourceInternal, at, rules.Change{})
	require.NoError(t, err)

	assert.Equal(t, rules.Snapshot{Request: "season-0:finale", Set: "rush", Source: rules.SourceInternal, AppliedAt: at},
		change.TakeSnapshot())
}

func TestARequestAlreadyKeptIsNotAppliedAgain(t *testing.T) {
	kept := rules.Snapshot{Request: "season-0:finale", Set: "rush", Source: rules.SourceInternal, AppliedAt: at}.Restore()

	again, err := rules.NewChange("season-0:finale", rules.Set{Name: "rush"}, rules.SourceInternal, at.Add(time.Hour), kept)

	require.ErrorIs(t, err, rules.ErrAppliedAlready)
	assert.Equal(t, kept, again, "the change kept stands")
}

func TestARequestNeedsAnIDOfAReasonableLength(t *testing.T) {
	_, err := rules.NewChange("", rules.Set{Name: "rush"}, rules.SourceOperator, at, rules.Change{})
	require.ErrorIs(t, err, rules.ErrNoRequest)

	_, err = rules.NewChange(rules.RequestID(strings.Repeat("é", 129)), rules.Set{Name: "rush"}, rules.SourceOperator, at,
		rules.Change{})
	require.ErrorIs(t, err, rules.ErrLongRequest)

	_, err = rules.NewChange(rules.RequestID(strings.Repeat("é", 128)), rules.Set{Name: "rush"}, rules.SourceOperator, at,
		rules.Change{})
	require.NoError(t, err)
}
