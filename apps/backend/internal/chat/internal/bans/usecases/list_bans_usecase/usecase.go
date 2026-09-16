// Package list_bans_usecase reads every chat ban a person passed.
package list_bans_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
)

type Reader interface {
	All() []bans.Ban
}

func New(reader Reader) *UseCase {
	return &UseCase{reader: reader}
}

type UseCase struct {
	reader Reader
}

func (u *UseCase) Execute(_ context.Context) []bans.Ban {
	return u.reader.All()
}
