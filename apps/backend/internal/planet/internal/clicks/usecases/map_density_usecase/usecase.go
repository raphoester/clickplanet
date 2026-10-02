package map_density_usecase

import "context"

type MaxIndexReader interface {
	MaxIndex() uint32
}

func New(tilesChecker MaxIndexReader) *UseCase {
	return &UseCase{tilesChecker: tilesChecker}
}

type UseCase struct {
	tilesChecker MaxIndexReader
}

func (u *UseCase) Execute(_ context.Context) uint32 {
	return u.tilesChecker.MaxIndex()
}
