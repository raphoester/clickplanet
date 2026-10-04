package click_handler

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/clickbudget"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/frozenmap"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

var callerErrors = []error{
	clicks.ErrUnknownCountry,
	clicks.ErrTileOutOfRange,
	clicks.ErrBonusesTogether,
}

func toConnect(err error, out click_usecase.Out) error {
	if errors.Is(err, clicks.ErrThrottled) {
		return throttled(err, out)
	}

	if errors.Is(err, tempo.ErrFrozen) {
		return frozenmap.Refusal(err)
	}

	for _, callerError := range callerErrors {
		if errors.Is(err, callerError) {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	return err
}

func throttled(err error, out click_usecase.Out) error {
	refusal := connect.NewError(connect.CodeResourceExhausted, err)
	if !out.Limited {
		return refusal
	}

	if detail, detailErr := connect.NewErrorDetail(clickbudget.Encode(out.Budget)); detailErr == nil {
		refusal.AddDetail(detail)
	}

	return refusal
}
