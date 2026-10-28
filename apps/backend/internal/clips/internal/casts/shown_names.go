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

func NewShownNames(usernames Usernames, optOuts OptOuts) ShownNames {
	return ShownNames{usernames: usernames, optOuts: optOuts}
}

// A player's username, unless they keep it out of the clips.
type ShownNames struct {
	usernames Usernames
	optOuts   OptOuts
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
	return names, nil
}
