package players

import (
	"context"
	"errors"
	"fmt"
	"regexp"
)

type GuestCode string

const GuestCodeLength = 6

var guestCodePattern = regexp.MustCompile(`^[0-9a-f]{6}$`)

var (
	ErrNoGuestCode      = errors.New("the account has no guest code")
	ErrGuestCodeTaken   = errors.New("another account holds this guest code")
	ErrInvalidGuestCode = errors.New("not a guest code")
)

func GuestCodeOf(value string) (GuestCode, error) {
	if !guestCodePattern.MatchString(value) {
		return "", fmt.Errorf("%w: %q", ErrInvalidGuestCode, value)
	}
	return GuestCode(value), nil
}

func DisplayNameOf(username Name, code GuestCode) string {
	if username != "" {
		return string(username)
	}
	return ReservedPrefix + string(code)
}

type CodeGenerator interface {
	NewGuestCode() (GuestCode, error)
}

type GuestCodeStore interface {
	GuestCode(ctx context.Context, account AccountID) (GuestCode, error)
	SaveGuestCode(ctx context.Context, account AccountID, code GuestCode) error
}

const maxDraws = 10

var ErrNoFreeGuestCode = errors.New("every guest code drawn was taken")

type GuestCodes struct {
	store     GuestCodeStore
	generator CodeGenerator
}

func NewGuestCodes(store GuestCodeStore, generator CodeGenerator) GuestCodes {
	return GuestCodes{store: store, generator: generator}
}

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
