package accounts

import "strings"

// The inbox an address delivers to: what one verified address is, for "one address is one player". Two addresses of
// one mailbox reach the same person, so they are one identity and find one account.
type Mailbox string

func (m Mailbox) String() string {
	return string(m)
}

const gmail = "gmail.com"

// The providers that deliver "name+anything" to "name", and allow no "+" in a name of their own, so the tag is cut
// whether they document it or not: an address that is not an alias there reaches nobody, and no code arrives. Only
// Gmail ignores the dots too. A domain added here changes the mailbox of addresses already kept: it comes with a
// migration that folds them again, as 20261115123255_identity_mailbox did.
var plusAddressed = map[string]bool{
	gmail: true, "googlemail.com": true,
	"outlook.com": true, "outlook.fr": true, "outlook.de": true, "outlook.es": true, "outlook.it": true,
	"hotmail.com": true, "hotmail.fr": true, "hotmail.co.uk": true, "hotmail.de": true, "hotmail.it": true,
	"hotmail.es": true, "live.com": true, "live.fr": true, "live.co.uk": true, "msn.com": true,
	"icloud.com": true, "me.com": true, "mac.com": true,
	"proton.me": true, "protonmail.com": true, "protonmail.ch": true, "pm.me": true,
	"fastmail.com": true, "fastmail.fm": true,
	"yandex.ru": true, "yandex.com": true, "ya.ru": true,
}

// The address in lower case, with what its provider ignores cut: the "+" and what follows it in the name, for the
// providers above, and Gmail's dots, with googlemail.com read as gmail.com. Any other address is its own mailbox: a
// "+" may be part of a name there, and a wrong guess would sign a stranger in to another player's account.
func MailboxOf(address string) Mailbox {
	lowered := strings.ToLower(strings.TrimSpace(address))
	at := strings.LastIndex(lowered, "@")
	if at < 0 {
		return Mailbox(lowered)
	}
	name, domain := lowered[:at], lowered[at+1:]
	if !plusAddressed[domain] {
		return Mailbox(lowered)
	}

	name, _, _ = strings.Cut(name, "+")
	if domain == gmail || domain == "googlemail.com" {
		name, domain = strings.ReplaceAll(name, ".", ""), gmail
	}
	if name == "" {
		return Mailbox(lowered)
	}
	return Mailbox(name + "@" + domain)
}
