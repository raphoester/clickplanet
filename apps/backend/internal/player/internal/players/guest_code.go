package players

import (
	"context"
	"errors"
	"fmt"
	"regexp"
)

// GuestCode is what the game calls an account that chose no username: 6 hex characters, drawn the first time
// the account is shown and kept until it is deleted. It is public. It says nothing about the account or the
// address, and it does not change when the player's network does.
type GuestCode string

// GuestCodeLength is how many hex characters a code has: 16.7M codes, each held by one account at most.
const GuestCodeLength = 6

var guestCodePattern = regexp.MustCompile(`^[0-9a-f]{6}$`)

var (
	// ErrNoGuestCode is an account never given a code.
	ErrNoGuestCode = errors.New("the account has no guest code")
	// ErrGuestCodeTaken is a code another account holds.
	ErrGuestCodeTaken = errors.New("another account holds this guest code")
	// ErrInvalidGuestCode is a code that is not 6 lowercase hex characters.
	ErrInvalidGuestCode = errors.New("not a guest code")
)

// GuestCodeOf checks a code: GuestCodeLength lowercase hex characters.
func GuestCodeOf(value string) (GuestCode, error) {
	if !guestCodePattern.MatchString(value) {
		return "", fmt.Errorf("%w: %q", ErrInvalidGuestCode, value)
	}
	return GuestCode(value), nil
}

// DisplayNameOf is the name the game shows for an account: its username, or ReservedPrefix and its guest code
// when it chose none. No username starts with the prefix, so a guest cannot pass for a player.
func DisplayNameOf(username Name, code GuestCode) string {
	if username != "" {
		return string(username)
	}
	return ReservedPrefix + string(code)
}

// CodeGenerator draws guest codes.
type CodeGenerator interface {
	NewGuestCode() (GuestCode, error)
}

// GuestCodeStore is where each account's code is kept.
type GuestCodeStore interface {
	// GuestCode answers ErrNoGuestCode when the account was never given one.
	GuestCode(ctx context.Context, account AccountID) (GuestCode, error)
	// SaveGuestCode gives the account code, unless it holds one already, which it then keeps. ErrGuestCodeTaken
	// when another account holds code.
	SaveGuestCode(ctx context.Context, account AccountID, code GuestCode) error
}

// maxDraws is how many codes GuestCodes.Assign draws before it gives up. With a million accounts holding one,
// a draw is taken one time in 17, so ten taken in a row is a generator that repeats itself.
const maxDraws = 10

// ErrNoFreeGuestCode is maxDraws codes drawn in a row, each one another account's.
var ErrNoFreeGuestCode = errors.New("every guest code drawn was taken")

// GuestCodes gives each account a guest code, once.
type GuestCodes struct {
	store     GuestCodeStore
	generator CodeGenerator
}

func NewGuestCodes(store GuestCodeStore, generator CodeGenerator) GuestCodes {
	return GuestCodes{store: store, generator: generator}
}

// Assign gives the account a code when it has none. A code another account holds is drawn again. Two calls at
// once for one account leave it with one code: the store keeps the first.
func (g GuestCodes) Assign(ctx context.Context, account AccountID) error {
	_, err := g.store.GuestCode(ctx, account)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrNoGuestCode) {
		return fmt.Errorf("failed to read the guest code: %w", err)
	}

	for range maxDraws {
		code, err := g.generator.NewGuestCode()
		if err != nil {
			return fmt.Errorf("failed to draw a guest code: %w", err)
		}

		err = g.store.SaveGuestCode(ctx, account, code)
		if errors.Is(err, ErrGuestCodeTaken) {
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to save the guest code: %w", err)
		}
		return nil
	}
	return ErrNoFreeGuestCode
}
