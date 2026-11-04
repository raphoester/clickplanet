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
	MuteError(ctx context.Context, caller mutes.Caller, kind mutes.Kind) error
}

func New(implementation UseCase, mutes Mutes) *Decorator {
	return &Decorator{implementation: implementation, mutes: mutes}
}

type Decorator struct {
	implementation UseCase
	mutes          Mutes
}

// A mute of the whole chat first: a player muted both ways is told the one that stops it from writing at all.
func (d *Decorator) Execute(ctx context.Context, in send_message_usecase.In) (messages.Message, error) {
	caller := mutes.NewCaller(in.Account, cpctx.GetSourceIP(ctx))
	if err := d.mutes.MuteError(ctx, caller, mutes.Chat); err != nil {
		return messages.Message{}, err //nolint:wrapcheck // the book named it, and a refusal keeps its type.
	}
	if !in.Gif.None() {
		if err := d.mutes.MuteError(ctx, caller, mutes.Gifs); err != nil {
			return messages.Message{}, err //nolint:wrapcheck // the book named it, and a refusal keeps its type.
		}
	}
	return d.implementation.Execute(ctx, in) //nolint:wrapcheck // a decorator adds a gate, not a sentence.
}
