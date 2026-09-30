//go:build testing

package postgres_event_store

import "github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"

// NewChunked is New with a chunk small enough for a test to cross it with a handful of rows.
func NewChunked(db cppg.QuerierBeginner, chunk int) *Store {
	return &Store{db: db, chunk: chunk}
}
