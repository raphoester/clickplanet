//go:build testing

package postgres_event_store

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"

func NewChunked(db cppg.QuerierBeginner, chunk int) *Store {
	return &Store{db: db, chunk: chunk}
}
