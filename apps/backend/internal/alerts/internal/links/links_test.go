package links_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
)

var (
	now     = time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)
	account = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))
)

func TestALinkFollowsTheFlagItsPlayerPledged(t *testing.T) {
	link := links.NewLink(account, "42", "", now)

	move, moved := link.Follow("ie")
	require.True(t, moved)
	assert.Equal(t, links.Move{User: "42", From: "", To: "ie"}, move)

	_, moved = link.Follow("ie")
	assert.False(t, moved)

	move, moved = link.Follow("gb")
	require.True(t, moved)
	assert.Equal(t, links.Move{User: "42", From: "ie", To: "gb"}, move)
	assert.Equal(t, nations.Flag("gb"), link.TakeSnapshot().Flag)
}

func TestAForgottenLinkTakesItsFlagsRoleOffTheUser(t *testing.T) {
	link := links.NewLink(account, "42", "ie", now)

	assert.Equal(t, links.Move{User: "42", From: "ie"}, link.Forget())
	assert.True(t, link.TakeSnapshot().Forgotten)
}

func TestAJoinAgainOfTheSameUserChangesNothing(t *testing.T) {
	link := links.NewLink(account, "42", "ie", now)

	assert.True(t, link.Joins(account, "42"))
	assert.False(t, link.Joins(account, "43"))
	assert.False(t, link.Joins(links.AccountID(uuid.MustParse("01a193de-df90-7637-9d85-7256584024a6")), "42"))
}

func TestAnAccountIsAUUID(t *testing.T) {
	_, err := links.AccountIDOf("not one")
	require.ErrorIs(t, err, links.ErrInvalidAccount)
	_, err = links.AccountIDOf(uuid.Nil.String())
	require.ErrorIs(t, err, links.ErrInvalidAccount)

	parsed, err := links.AccountIDOf(account.String())
	require.NoError(t, err)
	assert.Equal(t, account, parsed)
}
