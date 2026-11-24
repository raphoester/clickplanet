package renewalmessage

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	shopv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/shop/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

// Renewal is a renewal as its buyer reads it.
func Renewal(kept renewals.Snapshot) *shopv1.Renewal {
	renewal := &shopv1.Renewal{
		Id: kept.ID.String(), Plan: Plan(kept.Order.Plan), State: stateOf(kept.State),
		PeriodEndsAt: timestamppb.New(kept.PeriodEndsAt),
	}
	if kind, id := kept.Order.Target.Kept(); kind == orders.SubmissionTarget {
		renewal.Target = &shopv1.Renewal_SubmissionId{SubmissionId: id}
	} else {
		renewal.Target = &shopv1.Renewal_Flag{Flag: id}
	}
	return renewal
}

func Plan(plan plans.Plan) shopv1.Plan {
	switch plan {
	case plans.Week:
		return shopv1.Plan_PLAN_WEEK
	case plans.Month:
		return shopv1.Plan_PLAN_MONTH
	case plans.Year:
		return shopv1.Plan_PLAN_YEAR
	}
	return shopv1.Plan_PLAN_UNSPECIFIED
}

// PlanOf is the plan a page asked for, empty for none.
func PlanOf(plan shopv1.Plan) string {
	switch plan {
	case shopv1.Plan_PLAN_WEEK:
		return string(plans.Week)
	case shopv1.Plan_PLAN_MONTH:
		return string(plans.Month)
	case shopv1.Plan_PLAN_YEAR:
		return string(plans.Year)
	case shopv1.Plan_PLAN_UNSPECIFIED:
	}
	return ""
}

func stateOf(state renewals.State) shopv1.RenewalState {
	switch state {
	case renewals.Renewing:
		return shopv1.RenewalState_RENEWAL_STATE_RENEWING
	case renewals.Stopping:
		return shopv1.RenewalState_RENEWAL_STATE_STOPPING
	case renewals.Ended:
	}
	return shopv1.RenewalState_RENEWAL_STATE_UNSPECIFIED
}
