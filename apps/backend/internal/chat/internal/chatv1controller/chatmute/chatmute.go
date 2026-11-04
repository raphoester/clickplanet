package chatmute

import (
	"errors"
	"fmt"

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
	if detail, detailErr := connect.NewErrorDetail(&chatv1.MuteRefusal{
		MutedUntilUnixMs: refusal.Until().UnixMilli(),
		Kind:             Kind(refusal.Kind()),
	}); detailErr == nil {
		refused.AddDetail(detail)
	}
	return refused, true
}

func Kind(kind mutes.Kind) chatv1.MuteKind {
	if kind == mutes.Gifs {
		return chatv1.MuteKind_MUTE_KIND_GIFS
	}
	return chatv1.MuteKind_MUTE_KIND_CHAT
}

// A number the proto does not name is ErrInvalidKind, never the whole chat by default.
func KindOf(kind chatv1.MuteKind) (mutes.Kind, error) {
	switch kind {
	case chatv1.MuteKind_MUTE_KIND_CHAT:
		return mutes.Chat, nil
	case chatv1.MuteKind_MUTE_KIND_GIFS:
		return mutes.Gifs, nil
	default:
		return mutes.Chat, fmt.Errorf("%w: %d", mutes.ErrInvalidKind, kind)
	}
}
