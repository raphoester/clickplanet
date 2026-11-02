package links_test

import (
	"crypto/sha256"
	"encoding/base64"
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

func flow(t *testing.T) links.Flow {
	t.Helper()
	flow, err := links.NewFlow(account, &links.SequentialSecrets{}, now)
	require.NoError(t, err)
	return flow
}

func TestAFlowAsksForTheChallengeOfItsVerifier(t *testing.T) {
	authorization := flow(t).Authorization()

	sum := sha256.Sum256([]byte("secret-2"))
	assert.Equal(t, links.Authorization{State: "secret-1", Challenge: base64.RawURLEncoding.EncodeToString(sum[:])},
		authorization)
}

func TestAFlowIsSpentOnceWithItsStateBeforeItExpires(t *testing.T) {
	spent := flow(t)

	verifier, err := spent.Spend("secret-1", now.Add(links.FlowTTL-time.Second))
	require.NoError(t, err)
	assert.Equal(t, "secret-2", verifier)

	_, err = spent.Spend("secret-1", now)
	assert.ErrorIs(t, err, links.ErrFlowInvalid)
	assert.True(t, spent.TakeSnapshot().Spent)
}

func TestAFlowRefusesAnotherStateOrALateTrade(t *testing.T) {
	other := flow(t)
	_, err := other.Spend("secret-2", now)
	require.ErrorIs(t, err, links.ErrFlowInvalid)
	assert.False(t, other.TakeSnapshot().Spent)

	late := flow(t)
	_, err = late.Spend("secret-1", now.Add(links.FlowTTL))
	assert.ErrorIs(t, err, links.ErrFlowInvalid)
}

func TestALinkFollowsTheFlagItsPlayerTakesFor(t *testing.T) {
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

func TestAnAccountIsAUUID(t *testing.T) {
	_, err := links.AccountIDOf("not one")
	require.ErrorIs(t, err, links.ErrInvalidAccount)
	_, err = links.AccountIDOf(uuid.Nil.String())
	require.ErrorIs(t, err, links.ErrInvalidAccount)

	parsed, err := links.AccountIDOf(account.String())
	require.NoError(t, err)
	assert.Equal(t, account, parsed)
}
