package sessionv1controller

import (
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
)

// ErrRefused is the whole of what a refused caller is told. The reason — a
// forged token, a token for another site, siteverify being unreachable — is
// logged and goes no further.
var ErrRefused = errors.New("could not start a session")

// refusals are logged here rather than by the error net cpbootstrap wraps every
// service in, because that net logs at Error and a refused mint is not a fault
// of this server: it is the check doing its job, and on a public endpoint it is
// the common case. Anything the handler does not recognise still reaches the
// net, and is still logged as the error it is.
func toConnect(logger *slog.Logger, procedure string, err error) error {
	if !errors.Is(err, domain.ErrAttestationFailed) {
		return err
	}

	logger.Info("refused a session",
		slog.String("procedure", procedure),
		slog.Any("error", err),
	)

	return connect.NewError(connect.CodePermissionDenied, ErrRefused)
}
