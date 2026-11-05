package nickname_subscriber

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/subscribers"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

// What player tells of an account whose name it shows changed: player.v1.PlayerNamed, PlayerLeaving and PlayerReturned.
type Event interface {
	proto.Message
	GetAccountId() string
}

type UseCase interface {
	Execute(ctx context.Context, account links.AccountID) error
}

func New[T Event](useCase UseCase, logger *slog.Logger) Subscriber[T] {
	return Subscriber[T]{useCase: useCase, logger: logger}
}

type Subscriber[T Event] struct {
	useCase UseCase
	logger  *slog.Logger
}

// The event names the account alone, and the use case asks player its name now, so whichever of the three comes last
// leaves the nickname right. A member Discord will not rename (the server's owner, a member above the bot) is logged and
// not tried again: the next rename tries, as does the operator's LinkMembers. Anything else is tried again.
func (s Subscriber[T]) Handle(ctx context.Context, event T) error {
	account, err := links.AccountIDOf(event.GetAccountId())
	if err != nil {
		return err //nolint:wrapcheck // the rule named it.
	}
	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	err = s.useCase.Execute(ctx, account)
	if errors.Is(err, discord.ErrRefused) {
		s.logger.WarnContext(ctx, "Discord refused a player's nickname",
			slog.String("event", string(event.ProtoReflect().Descriptor().FullName())),
			slog.String("account", account.String()),
			slog.Any("error", err),
		)
		return nil
	}
	return err //nolint:wrapcheck // the use case named it.
}

var (
	_ cpredis.Handler[*playerv1.PlayerNamed]    = Subscriber[*playerv1.PlayerNamed]{}
	_ cpredis.Handler[*playerv1.PlayerLeaving]  = Subscriber[*playerv1.PlayerLeaving]{}
	_ cpredis.Handler[*playerv1.PlayerReturned] = Subscriber[*playerv1.PlayerReturned]{}
)
