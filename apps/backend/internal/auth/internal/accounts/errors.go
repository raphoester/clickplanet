package accounts

import "errors"

var (
	ErrNoSessionCookie  = errors.New("the browser sent no session cookie")
	ErrSessionNotFound  = errors.New("no session for this token")
	ErrSessionExpired   = errors.New("the session has expired")
	ErrNoAccount        = errors.New("this browser has no account")
	ErrAccountNotFound  = errors.New("no such account")
	ErrIdentityNotFound = errors.New("no account has this identity")
	ErrIdentityTaken    = errors.New("another account has this identity")
	// A link refused: the identity is another account's. Nothing is written.
	ErrIdentityLinkedElsewhere = errors.New("another account already uses this identity")
	// A link refused: the account already has another user of this provider. Nothing is written.
	ErrProviderAlreadyLinked = errors.New("this account already has a user of this provider")
)
