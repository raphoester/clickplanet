package notes

import (
	"errors"
	"fmt"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

var ErrNotANoteAddress = errors.New("the address names no note")

// The address the operator answers from. Each note has its own under it, which the player's reply comes back to.
type Inbox struct {
	local, domain string
}

func NewInbox(address cpmail.Address) Inbox {
	at := strings.LastIndex(string(address), "@")
	return Inbox{local: string(address)[:at], domain: address.Domain()}
}

func (i Inbox) Of(note NoteID) cpmail.Address {
	return cpmail.Address(i.local + "+" + note.String() + "@" + i.domain)
}

func (i Inbox) NoteOf(recipient string) (NoteID, error) {
	address, err := cpmail.AddressOf(recipient)
	if err != nil {
		return NoteID{}, fmt.Errorf("%w: %w", ErrNotANoteAddress, err)
	}
	local, detail, tagged := strings.Cut(strings.TrimSuffix(string(address), "@"+i.domain), "+")
	if !tagged || local != i.local || address.Domain() != i.domain {
		return NoteID{}, fmt.Errorf("%w: %q is not under %s@%s", ErrNotANoteAddress, recipient, i.local, i.domain)
	}
	id, err := NoteIDOf(detail)
	if err != nil {
		return NoteID{}, fmt.Errorf("%w: %w", ErrNotANoteAddress, err)
	}
	return id, nil
}
