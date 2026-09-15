package get_map_handler

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// toConnect leaves anything it does not recognise alone, for NewErrorInterceptor
// to log once and answer as an internal error.
func toConnect(err error) error {
	if errors.Is(err, clicks.ErrInvalidTileRange) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	return err
}
