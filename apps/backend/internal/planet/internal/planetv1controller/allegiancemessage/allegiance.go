package allegiancemessage

import (
	"errors"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
)

func Encode(flag string, changeableAt time.Time) *planetv1.Allegiance {
	allegiance := &planetv1.Allegiance{CountryId: flag}
	if !changeableAt.IsZero() {
		allegiance.ChangeableAtUnixMs = changeableAt.UnixMilli()
	}

	return allegiance
}

func Refusal(err error) (*connect.Error, bool) {
	var refusal allegiances.Refusal
	if !errors.As(err, &refusal) {
		return nil, false
	}

	refused := connect.NewError(connect.CodeFailedPrecondition, allegiances.ErrChangedTooSoon)
	if detail, detailErr := connect.NewErrorDetail(Encode(refusal.Flag, refusal.Until)); detailErr == nil {
		refused.AddDetail(detail)
	}

	return refused, true
}
