package graffiti_query

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	vandalsquadv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/vandalsquad/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
)

type Book interface {
	Entries() []graffiti.Graffiti
}

func NewMemoryQuery(book Book) *MemoryQuery {
	return &MemoryQuery{book: book}
}

type MemoryQuery struct {
	book Book
}

func (q *MemoryQuery) Graffiti() *vandalsquadv1.ListGraffitiResponse {
	entries := q.book.Entries()

	answer := &vandalsquadv1.ListGraffitiResponse{Graffiti: make([]*vandalsquadv1.Graffiti, len(entries))}
	for i, entry := range entries {
		tiles := make([]uint32, len(entry.Tiles))
		for j, tile := range entry.Tiles {
			tiles[j] = uint32(tile)
		}
		answer.Graffiti[i] = &vandalsquadv1.Graffiti{
			AccountId: entry.Account.String(),
			CountryId: entry.Flag,
			Kind:      kindOf(entry.Kind),
			TileIds:   tiles,
			FlaggedAt: timestamppb.New(entry.FlaggedAt),
			SeenAt:    timestamppb.New(entry.SeenAt),
			Link:      entry.Link,
		}
	}
	return answer
}

func kindOf(kind graffiti.Kind) vandalsquadv1.GraffitiKind {
	switch kind {
	case graffiti.Swastika:
		return vandalsquadv1.GraffitiKind_GRAFFITI_KIND_SWASTIKA
	default:
		return vandalsquadv1.GraffitiKind_GRAFFITI_KIND_UNSPECIFIED
	}
}
