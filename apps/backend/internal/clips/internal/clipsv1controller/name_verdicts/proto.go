// Package name_verdicts writes a verdict of a player's name as the operator's tools read it.
package name_verdicts

import (
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
)

func ProtoOf(verdict verdicts.Snapshot) (*clipsv1.NameVerdict, error) {
	by, err := byOf(verdict.By)
	if err != nil {
		return nil, err
	}
	return &clipsv1.NameVerdict{
		AccountId: verdict.Account.String(),
		Name:      verdict.Name,
		Shown:     verdict.Shown,
		Reason:    verdict.Reason,
		DecidedBy: by,
		DecidedAt: timestamppb.New(verdict.At),
	}, nil
}

func byOf(decider verdicts.Decider) (clipsv1.VerdictBy, error) {
	switch decider {
	case verdicts.ByJudge:
		return clipsv1.VerdictBy_VERDICT_BY_JUDGE, nil
	case verdicts.ByOperator:
		return clipsv1.VerdictBy_VERDICT_BY_OPERATOR, nil
	}
	return clipsv1.VerdictBy_VERDICT_BY_UNSPECIFIED, fmt.Errorf("%w: %q", verdicts.ErrUnknownDecider, decider)
}
