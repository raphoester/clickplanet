package shown_names_query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Usernames interface {
	// An account with no username, a guest's among them, is left out.
	Of(ctx context.Context, accounts []cpsession.AccountID) (map[cpsession.AccountID]string, error)
}

func NewPostgresQuery(db cppg.Conn, usernames Usernames) *PostgresQuery {
	return &PostgresQuery{db: db, usernames: usernames}
}

type PostgresQuery struct {
	db        cppg.Conn
	usernames Usernames
}

// A verdict of a name the player left shows nothing: the name must be the one it holds now.
const shown = `
	SELECT held.account, held.name
	FROM unnest($1::uuid[], $2::text[]) WITH ORDINALITY AS held(account, name, place)
	JOIN name_verdicts v ON v.account = held.account AND v.name = held.name
	WHERE v.shown AND NOT EXISTS (SELECT 1 FROM opt_outs o WHERE o.account = held.account)
	ORDER BY held.place
`

func (q *PostgresQuery) ShownNames(ctx context.Context, accounts []cpsession.AccountID) (*clipsv1.GetShownNamesResponse, error) {
	answer := &clipsv1.GetShownNamesResponse{}
	asked := onceEach(accounts)
	if len(asked) == 0 {
		return answer, nil
	}
	held, err := q.usernames.Of(ctx, asked)
	if err != nil {
		return nil, fmt.Errorf("failed to read the usernames the accounts hold: %w", err)
	}

	ids, names := make([]string, 0, len(held)), make([]string, 0, len(held))
	for _, account := range asked {
		if name, named := held[account]; named {
			ids, names = append(ids, account.String()), append(names, name)
		}
	}
	if len(ids) == 0 {
		return answer, nil
	}

	rows, err := q.db.On(ctx).QueryContext(ctx, shown, pq.Array(ids), pq.Array(names))
	if err != nil {
		return nil, fmt.Errorf("failed to read the names the posts may show: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			account uuid.UUID
			name    string
		)
		if err := rows.Scan(&account, &name); err != nil {
			return nil, fmt.Errorf("failed to read a name the posts may show: %w", err)
		}
		answer.ShownNames = append(answer.ShownNames, &clipsv1.ShownName{AccountId: account.String(), Name: name})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the names the posts may show: %w", err)
	}
	return answer, nil
}

func onceEach(accounts []cpsession.AccountID) []cpsession.AccountID {
	seen := cpcolls.NewSetWithCapacity[cpsession.AccountID](len(accounts))
	asked := make([]cpsession.AccountID, 0, len(accounts))
	for _, account := range accounts {
		if !seen.Contains(account) {
			seen.Add(account)
			asked = append(asked, account)
		}
	}
	return asked
}
