//go:build testing

package cpoutbox

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func Kept(ctx context.Context, db cppg.Querier) ([]proto.Message, error) {
	rows, err := db.QueryContext(ctx, `SELECT position, event::text, payload FROM outbox ORDER BY position`)
	if err != nil {
		return nil, fmt.Errorf("failed to read the outbox: %w", err)
	}
	_, events, unread, err := read(rows)
	if err != nil {
		return nil, err
	}
	if len(unread) > 0 {
		return nil, unread[0]
	}
	return events, nil
}
