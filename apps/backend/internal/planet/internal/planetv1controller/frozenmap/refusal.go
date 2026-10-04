package frozenmap

import (
	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

// The detail is what tells it apart: a refill on a full bank is FailedPrecondition too.
func Refusal(err error) *connect.Error {
	refusal := connect.NewError(connect.CodeFailedPrecondition, err)
	if detail, detailErr := connect.NewErrorDetail(&planetv1.MapFrozen{}); detailErr == nil {
		refusal.AddDetail(detail)
	}
	return refusal
}
