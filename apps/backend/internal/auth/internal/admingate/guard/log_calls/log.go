package log_calls

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/guard"
)

// Inside the guard: one line per call it let through, with who made it and what it asked, reads included.
func New(logger *slog.Logger) connect.Interceptor {
	return calls{logger: logger}
}

type calls struct {
	logger *slog.Logger
}

func (c calls) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
		response, err := next(ctx, request)
		c.log(ctx, request.Spec().Procedure, request.Any(), err)
		return response, err
	}
}

func (c calls) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (c calls) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		err := next(ctx, conn)
		c.log(ctx, conn.Spec().Procedure, nil, err)
		return err
	}
}

func (c calls) log(ctx context.Context, procedure string, message any, err error) {
	code := "ok"
	if err != nil {
		code = connect.CodeOf(err).String()
	}

	request := ""
	if asked, ok := message.(proto.Message); ok {
		if encoded, err := protojson.Marshal(asked); err == nil {
			request = string(encoded)
		}
	}

	c.logger.Info("admin call",
		slog.String("operator", string(guard.OperatorOf(ctx))),
		slog.String("procedure", procedure),
		slog.String("request", request),
		slog.String("code", code),
	)
}
