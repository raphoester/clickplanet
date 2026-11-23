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

func TestAReplyLosesTheMailItQuotesWhateverTheClient(t *testing.T) {
	for name, reply := range map[string]string{
		"gmail, the header on one line": "An iPhone 13.\r\nSafari.\r\n\r\nOn Mon, 23 Nov 2026 at 12:00, ClickPlanet <support+019a6f2e-1c2d-7e3f-8a9b-0c1d2e3f4a5b@clickplanet.lol> wrote:\r\n> Which phone?\r\n>\r\n> On 20 November 2026, you wrote:\r\n> > the globe is black\r\n",
		"gmail, the header wrapped":     "An iPhone 13.\nSafari.\n\nOn Mon, 23 Nov 2026 at 12:00, ClickPlanet <\nsupport+019a6f2e-1c2d-7e3f-8a9b-0c1d2e3f4a5b@clickplanet.lol> wrote:\n\n> Which phone?\n",
		"apple mail, in french":         "An iPhone 13.\nSafari.\n\nLe 23 nov. 2026 à 12:00, ClickPlanet <support+019a6f2e-1c2d-7e3f-8a9b-0c1d2e3f4a5b@ClickPlanet.lol> a écrit :\n\n> Which phone?",
		"outlook":                       "An iPhone 13.\nSafari.\n\n________________________________\nFrom: ClickPlanet <support+019a6f2e-1c2d-7e3f-8a9b-0c1d2e3f4a5b@clickplanet.lol>\nSent: Monday, 23 November 2026 12:00\nSubject: Re: your bug report\n\nWhich phone?",
		"quoted at the end alone":       "An iPhone 13.\nSafari.\n\n> Which phone?\n> \n",
		"nothing quoted":                "An iPhone 13.\nSafari.",
	} {
		assert.Equal(t, "An iPhone 13.\nSafari.", inbox.Unquoted(reply), name)
	}
}

func TestAReplyThatAnswersBetweenTheQuotedLinesKeepsThem(t *testing.T) {
	reply := "> Which phone?\nAn iPhone 13.\n> Which browser?\nSafari."

	assert.Equal(t, reply, inbox.Unquoted(reply))
}

func TestANoteWithNoAddressTakesNoReply(t *testing.T) {
	require.ErrorIs(t, note(t, notes.KindBug, "").AddressError(), notes.ErrNoReplyAddress)
	require.NoError(t, note(t, notes.KindBug, "ada@example.com").AddressError())
}
