package casts_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts"
)

type usernames map[casts.AccountID]string

func (u usernames) Of(context.Context, []casts.AccountID) (map[casts.AccountID]string, error) {
	return map[casts.AccountID]string(u), nil
}

func TestAPlayerWhoKeepsOutOfTheClipsHasNoNameShown(t *testing.T) {
	ada, bob := account(t, "01"), account(t, "02")
	optOuts := optouts.NewMemoryStore()
	require.NoError(t, optOuts.Keep(t.Context(), bob))

	shown, err := casts.NewShownNames(usernames{ada: "Ada", bob: "Bob"}, optOuts).Shown(t.Context(), []casts.AccountID{ada, bob})
	require.NoError(t, err)

	assert.Equal(t, map[casts.AccountID]string{ada: "Ada"}, shown)
}
