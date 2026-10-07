package chatmute

import (
	"errors"

	"connectrpc.com/connect"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

func Refusal(err error) (*connect.Error, bool) {
	var refusal mutes.Refusal
	if !errors.As(err, &refusal) {
		return nil, false
	}

	refused := connect.NewError(connect.CodePermissionDenied, mutes.ErrMuted)
	if detail, detailErr := connect.NewErrorDetail(&chatv1.MuteRefusal{MutedUntilUnixMs: refusal.Until().UnixMilli()}); detailErr == nil {
		refused.AddDetail(detail)
	}
	return refused, true
}
