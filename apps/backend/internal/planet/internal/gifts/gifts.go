package gifts

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

var ErrGiven = errors.New("the gift was already given to this account")

type Storage interface {
	Give(ctx context.Context, tag tempo.GiftTag, holder bonuses.Holder) error
}
