package casts

import (
	"context"
	"fmt"
)

type Usernames interface {
	// An account with no username, a guest's among them, is left out.
	Of(ctx context.Context, accounts []AccountID) (map[AccountID]string, error)
}

type OptOuts interface {
	Among(ctx context.Context, accounts []AccountID) ([]AccountID, error)
}

type Screening interface {
	// The names the platforms let the game's posts show.
	Execute(ctx context.Context, names map[AccountID]string) (map[AccountID]string, error)
}

func NewShownNames(usernames Usernames, optOuts OptOuts, screening Screening) ShownNames {
	return ShownNames{usernames: usernames, optOuts: optOuts, screening: screening}
}

// A player's username, unless they keep it out of the clips or the platforms forbid it.
type ShownNames struct {
	usernames Usernames
	optOuts   OptOuts
	screening Screening
}

func (n ShownNames) Shown(ctx context.Context, accounts []AccountID) (map[AccountID]string, error) {
	names, err := n.usernames.Of(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read the usernames: %w", err)
	}
	out, err := n.optOuts.Among(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read who keeps out of the clips: %w", err)
	}
	for _, account := range out {
		delete(names, account)
	}
	shown, err := n.screening.Execute(ctx, names)
	if err != nil {
		return nil, fmt.Errorf("failed to screen the names: %w", err)
	}
	return shown, nil
}
