package anonymize_recruitments_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
)

func New(kept arrivals.Recruitments) *UseCase {
	return &UseCase{recruitments: kept}
}

type UseCase struct {
	recruitments arrivals.Recruitments
}

func (u *UseCase) Execute(ctx context.Context, recruiter arrivals.AccountID) error {
	found, err := u.recruitments.By(ctx, recruiter)
	if err != nil {
		return fmt.Errorf("failed to read the recruiter's recruitments: %w", err)
	}

	if err := u.recruitments.Save(ctx, arrivals.AnonymizedRecruitments(found)...); err != nil {
		return fmt.Errorf("failed to anonymize the recruiter's recruitments: %w", err)
	}
	return nil
}
