package accounts

import (
	"slices"
	"time"
)

type Account struct {
	id         AccountID
	createdAt  time.Time
	identities []Identity
}

func AccountOf(id AccountID, createdAt time.Time, identities []Identity) *Account {
	return &Account{id: id, createdAt: createdAt, identities: identities}
}

func (a *Account) ID() AccountID {
	return a.id
}

func (a *Account) Providers() []string {
	providers := make([]string, 0, len(a.identities))
	for _, identity := range a.identities {
		providers = append(providers, identity.provider)
	}
	return providers
}

func (a *Account) linkedTo(provider string) bool {
	return slices.Contains(a.Providers(), provider)
}

type Identity struct {
	provider      string
	subject       string
	account       AccountID
	email         string
	emailVerified bool
	linkedAt      time.Time
}

func NewIdentity(provider string, claim Claim, account AccountID, now time.Time) *Identity {
	identity := &Identity{provider: provider, subject: claim.subject, account: account, linkedAt: now}
	if email := claim.VerifiedEmail(); email != "" {
		identity.email = email
		identity.emailVerified = true
	}
	return identity
}

func IdentityOf(provider string, subject string, account AccountID, email string, emailVerified bool, linkedAt time.Time) *Identity {
	return &Identity{
		provider: provider, subject: subject, account: account, email: email, emailVerified: emailVerified, linkedAt: linkedAt,
	}
}

func (i *Identity) Provider() string {
	return i.provider
}

func (i *Identity) Subject() string {
	return i.subject
}

func (i *Identity) Account() AccountID {
	return i.account
}

func (i *Identity) Email() string {
	return i.email
}

func (i *Identity) EmailVerified() bool {
	return i.emailVerified
}

func (i *Identity) LinkedAt() time.Time {
	return i.linkedAt
}

type Claim struct {
	subject       string
	email         string
	emailVerified bool
}

func ClaimOf(subject string, email string, emailVerified bool) Claim {
	return Claim{subject: subject, email: email, emailVerified: emailVerified}
}

func (c Claim) Subject() string {
	return c.subject
}

func (c Claim) VerifiedEmail() string {
	if !c.emailVerified {
		return ""
	}
	return c.email
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
	case joinable && current != nil && current.id == owner.id:
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
	case known != nil && known.account == current.id:
		return SignedIn, nil
	case known != nil:
		return 0, ErrIdentityLinkedElsewhere
	case owner != nil && owner.id != current.id:
		return 0, ErrIdentityLinkedElsewhere
	case current.linkedTo(provider):
		return 0, ErrProviderAlreadyLinked
	default:
		return Linked, nil
	}
}

type SignIn struct {
	newAccount bool
	identity   *Identity
	session    *Session
	replaces   TokenHash
}

func NewSignIn(session *Session) SignIn {
	return SignIn{session: session}
}

func (s SignIn) WithNewAccount() SignIn {
	s.newAccount = true
	return s
}

func (s SignIn) WithIdentity(identity *Identity) SignIn {
	s.identity = identity
	return s
}

func (s SignIn) WithReplaced(replaces TokenHash) SignIn {
	s.replaces = replaces
	return s
}

func (s SignIn) NewAccount() bool {
	return s.newAccount
}

func (s SignIn) Identity() *Identity {
	return s.identity
}

func (s SignIn) Session() *Session {
	return s.session
}

func (s SignIn) Replaces() TokenHash {
	return s.replaces
}
