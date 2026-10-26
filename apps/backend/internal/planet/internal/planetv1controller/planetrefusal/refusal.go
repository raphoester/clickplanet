package planetrefusal

import (
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func Refusal(err error) (*connect.Error, bool) {
	var refusal clicks.Refusal
	if !errors.As(err, &refusal) {
		return nil, false
	}

	refused := connect.NewError(connect.CodePermissionDenied, clicks.ErrRefused)
	if detail, detailErr := connect.NewErrorDetail(&planetv1.BanRefusal{BannedUntilUnixMs: refusal.Until().UnixMilli()}); detailErr == nil {
		refused.AddDetail(detail)
	}
	return refused, true
}
