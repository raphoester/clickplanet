// Package unban_member_usecase lifts a chat ban.
package unban_member_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type Unbanner interface {
	Unban(ctx context.Context, tag string) error
}

type In struct {
	AuthorTag string
}

type Out struct {
	AuthorTag string
}

func New(unbanner Unbanner) *UseCase {
	return &UseCase{unbanner: unbanner}
}

type UseCase struct {
	unbanner Unbanner
}

// Execute broadcasts nothing: the text is served again from the next history
// read, and un-blanking an open screen would mean re-sending what it blanked.
func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	tag, err := messages.ParseTag(in.AuthorTag)
	if err != nil {
		return Out{}, fmt.Errorf("%w: %q", err, in.AuthorTag)
	}

	if err := u.unbanner.Unban(ctx, tag); err != nil {
		return Out{}, fmt.Errorf("failed to lift the chat ban: %w", err)
	}

	return Out{AuthorTag: tag}, nil
}
