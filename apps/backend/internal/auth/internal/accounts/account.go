package accounts

import (
	"slices"
	"time"
)

// Account is a player, and the providers it signs in with. A guest has none.
type Account struct {
	ID AccountID
	// Oldest link first.
	Identities []Identity
}

func (a *Account) Linked() bool {
	return len(a.Identities) > 0
}

// Providers is the name of each linked provider, oldest link first.
func (a *Account) Providers() []string {
	providers := make([]string, 0, len(a.Identities))
	for _, identity := range a.Identities {
		providers = append(providers, identity.Provider)
	}
	return providers
}

func (a *Account) holds(provider string) bool {
	return slices.Contains(a.Providers(), provider)
}

// Identity is one provider's user, linked to one account.
type Identity struct {
	Provider string
	Subject  string
	Account  AccountID
	// Empty unless the provider said the address is verified.
	Email         string
	EmailVerified bool
	LinkedAt      time.Time
}

// Claim is what a provider says about the user who signed in.
type Claim struct {
	Subject       string
	Email         string
	EmailVerified bool
}

// NewIdentity links claim to account. An unverified email is dropped: nobody may be reached, or matched, on an address they may not own.
func NewIdentity(provider string, claim Claim, account AccountID, now time.Time) *Identity {
	identity := &Identity{Provider: provider, Subject: claim.Subject, Account: account, LinkedAt: now}
	if claim.EmailVerified && claim.Email != "" {
		identity.Email = claim.Email
		identity.EmailVerified = true
	}
	return identity
}

// Outcome is what a sign-in does with an identity.
type Outcome int

const (
	// The identity is known: the browser moves to its account, and the account it was on is left as it was.
	SignedIn Outcome = iota + 1
	// The identity is new: it is linked to the account the browser is on.
	Linked
	// The identity is new, and the browser has no account to take it: a new account is made.
	Created
)

// Choose decides a sign-in. Emails are never compared: two identities are one player only when the player links them.
func Choose(current *Account, known *Identity, provider string) Outcome {
	switch {
	case known != nil:
		return SignedIn
	case current == nil || current.holds(provider):
		return Created
	default:
		return Linked
	}
}

// SignIn is everything a sign-in writes, in one transaction.
type SignIn struct {
	// The account row to create first, when the outcome is Created.
	NewAccount bool
	// The identity to link, or nil when it was already known.
	Identity *Identity
	Session  *Session
	// The browser's previous session, deleted, or nil when it had none.
	Replaces TokenHash
}
