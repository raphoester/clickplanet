package notes_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var (
	inbox = notes.NewInbox("support@clickplanet.lol")
	id    = notes.NoteID(uuid.MustParse("019a6f2e-1c2d-7e3f-8a9b-0c1d2e3f4a5b"))
)

func TestEachNoteHasAnAddressUnderTheInboxThatLeadsBackToIt(t *testing.T) {
	address := inbox.Of(id)

	assert.Equal(t, cpmail.Address("support+019a6f2e-1c2d-7e3f-8a9b-0c1d2e3f4a5b@clickplanet.lol"), address)
	back, err := inbox.NoteOf(" Support+019A6F2E-1C2D-7E3F-8A9B-0C1D2E3F4A5B@ClickPlanet.lol ")
	require.NoError(t, err)
	assert.Equal(t, id, back)
}

func TestAnAddressThatNamesNoNoteUnderTheInboxIsRefused(t *testing.T) {
	for _, recipient := range []string{
		"support@clickplanet.lol",
		"support+@clickplanet.lol",
		"support+not-a-note@clickplanet.lol",
		"noreply+019a6f2e-1c2d-7e3f-8a9b-0c1d2e3f4a5b@clickplanet.lol",
		"support+019a6f2e-1c2d-7e3f-8a9b-0c1d2e3f4a5b@example.com",
		"Support <support+019a6f2e-1c2d-7e3f-8a9b-0c1d2e3f4a5b@clickplanet.lol>",
		"",
	} {
		_, err := inbox.NoteOf(recipient)

		assert.ErrorIs(t, err, notes.ErrNotANoteAddress, recipient)
	}
}
