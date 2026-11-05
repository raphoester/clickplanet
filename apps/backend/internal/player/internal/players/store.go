package players

import (
	"context"
	"errors"
)

var (
	ErrNoProfile = errors.New("the account has no profile")
	ErrNameTaken = errors.New("another player has this name")

	ErrProfileExists = errors.New("the account already has a profile")
)

type Store interface {
	GuestCodeStore

	Profile(ctx context.Context, account AccountID) (Profile, error)
	SaveProfile(ctx context.Context, profile Profile) error
	CreateProfile(ctx context.Context, profile Profile) error
	SaveColor(ctx context.Context, account AccountID, color Color) error
	DeleteAccount(ctx context.Context, account AccountID) error
}
