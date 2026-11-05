package chatmute

import (
	"errors"

	"connectrpc.com/connect"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/proto_kind"
)

func Refusal(err error) (*connect.Error, bool) {
	var refusal mutes.Refusal
	if !errors.As(err, &refusal) {
		return nil, false
	}

	refused := connect.NewError(connect.CodePermissionDenied, mutes.ErrMuted)
	if detail, detailErr := connect.NewErrorDetail(&chatv1.MuteRefusal{
		MutedUntilUnixMs: refusal.Until().UnixMilli(),
		Kind:             proto_kind.Encode(refusal.Kind()),
	}); detailErr == nil {
		refused.AddDetail(detail)
	}
	return refused, true
}
