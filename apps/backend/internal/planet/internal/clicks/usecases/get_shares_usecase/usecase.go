package get_shares_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type HoldingsReader interface {
	Holdings() []clicks.Holding
}

type MaxIndexReader interface {
	MaxIndex() uint32
}

type Out struct {
	MapTiles uint32
	Holdings []clicks.Holding
}

func New(holdings HoldingsReader, board MaxIndexReader) *UseCase {
	return &UseCase{holdings: holdings, board: board}
}

type UseCase struct {
	holdings HoldingsReader
	board    MaxIndexReader
}

func (u *UseCase) Execute(context.Context) Out {
	return Out{MapTiles: u.board.MaxIndex(), Holdings: u.holdings.Holdings()}
}
