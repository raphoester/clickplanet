package muting_send_message

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/send_message_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in send_message_usecase.In) (messages.Message, error)
}

type Mutes interface {
	MuteError(ctx context.Context, caller mutes.Caller) error
}

func New(implementation UseCase, mutes Mutes) *Decorator {
	return &Decorator{implementation: implementation, mutes: mutes}
}

type Decorator struct {
	implementation UseCase
	mutes          Mutes
}

func (d *Decorator) Execute(ctx context.Context, in send_message_usecase.In) (messages.Message, error) {
	if err := d.mutes.MuteError(ctx, mutes.NewCaller(in.Account, cpctx.GetSourceIP(ctx))); err != nil {
		return messages.Message{}, err //nolint:wrapcheck // the book named it, and a refusal keeps its type.
	}
	return d.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a gate, not a sentence.
}
