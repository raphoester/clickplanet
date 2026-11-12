package links_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
)

var (
	now     = time.Date(2026, 11, 2, 18, 0, 0, 0, time.UTC)
	account = links.AccountID(uuid.MustParse("01a19416-ad77-72b3-8c34-8bbbaec97d8b"))
)

func TestANewLinkGaveNoNameYet(t *testing.T) {
	link := links.NewLink(account, "42", "ie", now)

	_, renamed := link.Rename("")
	assert.False(t, renamed)
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
