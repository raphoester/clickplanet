package proto_kind

import (
	"fmt"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

// The kind on the wire: the admin calls, the refusal and the event say it the same way.
func Encode(kind mutes.Kind) chatv1.MuteKind {
	if kind == mutes.Gifs {
		return chatv1.MuteKind_MUTE_KIND_GIFS
	}
	return chatv1.MuteKind_MUTE_KIND_CHAT
}

// A number the proto does not name is ErrInvalidKind, never the whole chat by default.
func Decode(kind chatv1.MuteKind) (mutes.Kind, error) {
	switch kind {
	case chatv1.MuteKind_MUTE_KIND_CHAT:
		return mutes.Chat, nil
	case chatv1.MuteKind_MUTE_KIND_GIFS:
		return mutes.Gifs, nil
	default:
		return mutes.Chat, fmt.Errorf("%w: %d", mutes.ErrInvalidKind, kind)
	}
}
