package click_handler

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/adapters/primary/http/planetv1controller/clickbudget"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
)

// callerErrors are the refusals this procedure earns a 400 for. Two different
// mistakes landing on one code is the edge's business: the use case still says
// which, and the message it wrote travels with it.
var callerErrors = []error{
	clicks.ErrUnknownCountry,
	clicks.ErrTileOutOfRange,
}

// toConnect leaves anything it does not recognise alone, for NewErrorInterceptor
// to log once and answer as an internal error.
func toConnect(err error, out click.Out) error {
	if errors.Is(err, clicks.ErrThrottled) {
		return throttled(err, out)
	}

	for _, callerError := range callerErrors {
		if errors.Is(err, callerError) {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	return err
}

// throttled carries the reading as an error detail, because a refusal has no
// response message to put it in — and it is the answer a client most needs to
// read, since it says when the next token lands.
func throttled(err error, out click.Out) error {
	refusal := connect.NewError(connect.CodeResourceExhausted, err)
	if !out.Limited {
		return refusal
	}

	// Only a message that will not marshal fails here, which a generated one
	// does not. The caller has to be refused either way, so it is the detail
	// that is dropped rather than the refusal that becomes an error.
	if detail, detailErr := connect.NewErrorDetail(clickbudget.Encode(out.Budget)); detailErr == nil {
		refusal.AddDetail(detail)
	}

	return refusal
}
