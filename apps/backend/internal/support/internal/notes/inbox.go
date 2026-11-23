package notes

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
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

var outlook = regexp.MustCompile(`(?i)^(-{2,}\s*original message\s*-{2,}|_{10,})$`)

// A reply with the mails it quotes cut off: from the first line that names the inbox (the quote's header, as
// "On …, ClickPlanet <support+…@clickplanet.lol> wrote:" or "From: …"), and the quoted lines at its end.
func (i Inbox) Unquoted(reply string) string {
	lines := strings.Split(strings.ReplaceAll(reply, "\r\n", "\n"), "\n")
	marker := "@" + i.domain

	for n, line := range lines {
		trimmed := strings.TrimSpace(line)
		if outlook.MatchString(trimmed) {
			lines = lines[:n]
			break
		}
		if strings.Contains(strings.ToLower(trimmed), marker) {
			cut := n
			if n > 0 && strings.TrimSpace(lines[n-1]) != "" && (n == 1 || strings.TrimSpace(lines[n-2]) == "") {
				cut = n - 1
			}
			lines = lines[:cut]
			break
		}
	}

	for len(lines) > 0 {
		last := strings.TrimSpace(lines[len(lines)-1])
		if last != "" && !strings.HasPrefix(last, ">") && !outlook.MatchString(last) {
			break
		}
		lines = slices.Delete(lines, len(lines)-1, len(lines))
	}
	return strings.Join(lines, "\n")
}
