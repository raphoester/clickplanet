// Package ban_member_usecase silences one chat member and blanks what they said.
package ban_member_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Banner interface {
	Ban(ctx context.Context, ban bans.Ban) (bans.Ban, error)
}

// Log blanks the author on every open screen, and says how many of the messages it still holds were theirs.
type Log interface {
	Redact(tag string) int
}

type In struct {
	// AuthorTag is the tag as the chat shows it; the '#' is optional.
	AuthorTag string
	Reason    string
}

type Out struct {
	Ban      bans.Ban
	Redacted int
}

func New(banner Banner, log Log, clock cptime.Clock) *UseCase {
	return &UseCase{banner: banner, log: log, clock: clock}
}

type UseCase struct {
	banner Banner
	log    Log
	clock  cptime.Clock
}

// Execute records the ban before it broadcasts one: a redaction every screen
// obeyed but no table remembers is one a restart undoes.
func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	tag, err := messages.ParseTag(in.AuthorTag)
	if err != nil {
		return Out{}, fmt.Errorf("%w: %q", err, in.AuthorTag)
	}

	ban, err := u.banner.Ban(ctx, bans.Ban{AuthorTag: tag, BannedAt: u.clock.Now(), Reason: in.Reason})
	if err != nil {
		return Out{}, fmt.Errorf("failed to ban the chat member: %w", err)
	}

	return Out{Ban: ban, Redacted: u.log.Redact(tag)}, nil
}
