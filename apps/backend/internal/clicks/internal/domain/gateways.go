package domain

import "context"

type TilesChecker interface {
	CheckTile(tile uint32) bool
	MaxIndex() uint32
}

type TileStorage interface {
	Set(ctx context.Context, tile uint32, value string) error
}

type DenseBatch struct {
	Start uint32
	Codes []string
	Tiles []byte
}

type TileReporter interface {
	Subscribe(ctx context.Context) <-chan TileUpdate
}

type CountryChecker interface {
	CheckCountry(country string) bool
}
