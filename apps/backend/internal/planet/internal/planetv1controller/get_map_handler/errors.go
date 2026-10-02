package get_map_handler

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func toConnect(err error) error {
	if errors.Is(err, clicks.ErrInvalidTileRange) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	return err
}
