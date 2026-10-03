package accounts

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

type Account struct {
	ID         AccountID
	CreatedAt  time.Time
	Identities []Identity
}

func (a *Account) Linked() bool {
	return len(a.Identities) > 0
}

func (a *Account) Providers() []string {
	providers := make([]string, 0, len(a.Identities))
	for _, identity := range a.Identities {
		providers = append(providers, identity.Provider)
	}
	return providers
}

// The address the player typed comes before the ones a provider answered.
var emailRanks = map[string]int{"email": 0, "google": 1, "discord": 2}

func (a *Account) Emails() []string {
	identities := slices.Clone(a.Identities)
	slices.SortStableFunc(identities, func(x, y Identity) int {
		return cmp.Compare(emailRankOf(x.Provider), emailRankOf(y.Provider))
	})

	emails := []string{}
	for _, identity := range identities {
		if !identity.EmailVerified || identity.Email == "" {
			continue
		}
		if slices.ContainsFunc(emails, func(email string) bool { return strings.EqualFold(email, identity.Email) }) {
			continue
		}
		emails = append(emails, identity.Email)
	}
	return emails
}

func emailRankOf(provider string) int {
	if rank, ranked := emailRanks[provider]; ranked {
		return rank
	}
	return len(emailRanks)
}

func (a *Account) linkedTo(provider string) bool {
	return slices.Contains(a.Providers(), provider)
}

type Identity struct {
	Provider      string
	Subject       string
	Account       AccountID
	Email         string
	EmailVerified bool
	LinkedAt      time.Time
}

type Claim struct {
	Subject       string
	Email         string
	EmailVerified bool
}

func (c Claim) VerifiedEmail() string {
	if !c.EmailVerified {
		return ""
	}
	return c.Email
}

func NewIdentity(provider string, claim Claim, account AccountID, now time.Time) *Identity {
	identity := &Identity{Provider: provider, Subject: claim.Subject, Account: account, LinkedAt: now}
	if email := claim.VerifiedEmail(); email != "" {
		identity.Email = email
		identity.EmailVerified = true
	}
	return identity
}

type Intent int

const (
	// Must stay the zero value: flows sealed before intents existed carry none.
	IntentSignIn Intent = iota
	IntentLink
)

type Outcome int

const (
	SignedIn Outcome = iota + 1
	Linked
	Created
	Joined
)

func OutcomeOf(intent Intent, current *Account, known *Identity, owner *Account, provider string) (Outcome, error) {
	if intent == IntentLink {
		return linkOutcomeOf(current, known, owner, provider)
	}
	joinable := owner != nil && !owner.linkedTo(provider)
	switch {
	case known != nil:
		return SignedIn, nil
	case joinable && current != nil && current.ID == owner.ID:
		return Linked, nil
	case joinable:
		return Joined, nil
	case current == nil || current.linkedTo(provider):
		return Created, nil
	default:
		return Linked, nil
	}
}

func linkOutcomeOf(current *Account, known *Identity, owner *Account, provider string) (Outcome, error) {
	switch {
	case current == nil:
		return 0, ErrNoAccount
	case known != nil && known.Account == current.ID:
		return SignedIn, nil
	case known != nil:
		return 0, ErrIdentityLinkedElsewhere
	case owner != nil && owner.ID != current.ID:
		return 0, ErrIdentityLinkedElsewhere
	case current.linkedTo(provider):
		return 0, ErrProviderAlreadyLinked
	default:
		return Linked, nil
	}
}

type SignIn struct {
	NewAccount bool
	Identity   *Identity
	Session    *Session
	Replaces   TokenHash
}
