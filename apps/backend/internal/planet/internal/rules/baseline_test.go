package rules_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

func TestANewRequestIsABaselineOfItsSet(t *testing.T) {
	baseline, err := rules.NewBaseline("season-0:finale", setNamed(t, "rush"), at, rules.Baseline{})
	require.NoError(t, err)

	assert.Equal(t, rules.BaselineSnapshot{Request: "season-0:finale", Set: "rush", AppliedAt: at},
		baseline.TakeSnapshot())
}

func TestARequestAlreadyKeptIsNotAppliedAgain(t *testing.T) {
	kept := rules.BaselineSnapshot{Request: "season-0:finale", Set: "rush", AppliedAt: at}.Restore()

	again, err := rules.NewBaseline("season-0:finale", setNamed(t, rules.Default), at.Add(time.Hour), kept)

	require.ErrorIs(t, err, rules.ErrAppliedAlready)
	assert.Equal(t, kept, again, "the baseline kept stands")
}

func TestARequestNeedsAnIDOfAReasonableLength(t *testing.T) {
	_, err := rules.NewBaseline("", setNamed(t, "rush"), at, rules.Baseline{})
	require.ErrorIs(t, err, rules.ErrNoRequest)

	_, err = rules.NewBaseline(rules.RequestID(strings.Repeat("é", 129)), setNamed(t, "rush"), at, rules.Baseline{})
	require.ErrorIs(t, err, rules.ErrLongRequest)

	_, err = rules.NewBaseline(rules.RequestID(strings.Repeat("é", 128)), setNamed(t, "rush"), at, rules.Baseline{})
	require.NoError(t, err)
}
