package rpc_payables

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
)

var (
	ErrUnknown    = errors.New("the flags module keeps no such submission or flag")
	ErrNotYours   = errors.New("the submission or the flag is another account's")
	ErrNotPayable = errors.New("no time is sold for it now")
)

type Flags interface {
	GetPayable(
		ctx context.Context, req *connect.Request[flagsv1.GetPayableRequest],
	) (*connect.Response[flagsv1.GetPayableResponse], error)
}

func New(flags Flags) Payables {
	return Payables{flags: flags}
}

// Payables asks the flags module, in its words, whether an account may buy time for a submission or a flag.
type Payables struct {
	flags Flags
}

// Payable is every target whose time goes to the same flag, when the buyer may pay for the target.
func (p Payables) Payable(ctx context.Context, buyer orders.AccountID, target orders.Target) ([]orders.Target, error) {
	asked := &flagsv1.TimeTarget{}
	if kind, id := target.Kept(); kind == orders.SubmissionTarget {
		asked.Target = &flagsv1.TimeTarget_SubmissionId{SubmissionId: id}
	} else {
		asked.Target = &flagsv1.TimeTarget_Flag{Flag: id}
	}
	answer, err := p.flags.GetPayable(ctx, connect.NewRequest(&flagsv1.GetPayableRequest{
		AccountId: buyer.String(), Target: asked,
	}))
	switch connect.CodeOf(err) {
	case connect.CodeNotFound, connect.CodeInvalidArgument:
		return nil, fmt.Errorf("%w: %w", ErrUnknown, err)
	case connect.CodePermissionDenied:
		return nil, fmt.Errorf("%w: %w", ErrNotYours, err)
	case connect.CodeFailedPrecondition:
		return nil, fmt.Errorf("%w: %w", ErrNotPayable, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to ask the flags module: %w", err)
	}
	same := make([]orders.Target, 0, len(answer.Msg.GetSameFlag()))
	for _, told := range answer.Msg.GetSameFlag() {
		kept, err := orders.ParseTarget(told.GetSubmissionId(), told.GetFlag())
		if err != nil {
			return nil, fmt.Errorf("the flags module named a target the shop cannot keep: %w", err)
		}
		same = append(same, kept)
	}
	return same, nil
}
