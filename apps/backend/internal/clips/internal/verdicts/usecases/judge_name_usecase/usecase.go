package judge_name_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
)

type Usernames interface {
	// An account with no username, a guest's among them, is left out.
	Of(ctx context.Context, accounts []verdicts.AccountID) (map[verdicts.AccountID]string, error)
}

type Screening interface {
	Execute(ctx context.Context, names map[verdicts.AccountID]string) (map[verdicts.AccountID]string, error)
}

func New(usernames Usernames, screening Screening) *UseCase {
	return &UseCase{usernames: usernames, screening: screening}
}

type UseCase struct {
	usernames Usernames
	screening Screening
}

// The name the account holds now is judged, unless a verdict of it is kept already.
func (u *UseCase) Execute(ctx context.Context, account verdicts.AccountID) error {
	names, err := u.usernames.Of(ctx, []verdicts.AccountID{account})
	if err != nil {
		return fmt.Errorf("failed to read the account's username: %w", err)
	}
	if _, err := u.screening.Execute(ctx, names); err != nil {
		return fmt.Errorf("failed to judge the account's username: %w", err)
	}
	return nil
}
