package cpconnect

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
)

type Mapper func(error) *connect.Error

func NewErrorInterceptor(logger *slog.Logger, mapper Mapper) connect.Interceptor {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	return &errorInterceptor{logger: logger, mapper: mapper}
}

type errorInterceptor struct {
	logger *slog.Logger
	mapper Mapper
}

func (i *errorInterceptor) translate(procedure string, err error) error {
	if err == nil {
		return nil
	}

	if errors.As(err, new(*connect.Error)) {
		return err
	}

	if i.mapper != nil {
		if mapped := i.mapper(err); mapped != nil {
			return mapped
		}
	}

	i.logger.Error("rpc failed",
		slog.String("procedure", procedure),
		slog.Any("error", err),
	)

	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}

func (i *errorInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		res, err := next(ctx, req)
		if err != nil {
			return nil, i.translate(req.Spec().Procedure, err)
		}

		return res, nil
	}
}

func (i *errorInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return i.translate(conn.Spec().Procedure, next(ctx, conn))
	}
}

func (i *errorInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}
