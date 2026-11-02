package postgres_link_store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func New(db cppg.Conn) *Store {
	return &Store{db: db}
}

type Store struct {
	db cppg.Conn
}

var _ links.Store = (*Store)(nil)

func (s *Store) FlowOf(ctx context.Context, account links.AccountID) (links.Flow, error) {
	kept := links.FlowSnapshot{Account: account}
	err := s.db.On(ctx).QueryRowContext(ctx, `
		SELECT state, verifier, expires_at FROM link_flows WHERE account_id = $1
	`, uuid.UUID(account)).Scan(&kept.State, &kept.Verifier, &kept.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return links.Flow{}, links.ErrNoFlow
	}
	if err != nil {
		return links.Flow{}, fmt.Errorf("failed to read the link flow: %w", err)
	}
	kept.ExpiresAt = kept.ExpiresAt.UTC()
	return kept.Restore(), nil
}

func (s *Store) SaveFlow(ctx context.Context, flow links.Flow) error {
	kept := flow.TakeSnapshot()
	if kept.Spent {
		if _, err := s.db.On(ctx).ExecContext(ctx, `DELETE FROM link_flows WHERE account_id = $1`,
			uuid.UUID(kept.Account)); err != nil {
			return fmt.Errorf("failed to delete a spent link flow: %w", err)
		}
		return nil
	}
	if _, err := s.db.On(ctx).ExecContext(ctx, `
		INSERT INTO link_flows (account_id, state, verifier, expires_at) VALUES ($1, $2, $3, $4)
		ON CONFLICT (account_id) DO UPDATE SET
			state = EXCLUDED.state, verifier = EXCLUDED.verifier, expires_at = EXCLUDED.expires_at
	`, uuid.UUID(kept.Account), kept.State, kept.Verifier, kept.ExpiresAt.UTC()); err != nil {
		return fmt.Errorf("failed to upsert a link flow: %w", err)
	}
	return nil
}

func (s *Store) LinkOf(ctx context.Context, account links.AccountID) (links.Link, error) {
	return s.restored(ctx, `
		SELECT account_id, discord_user, flag, linked_at FROM links WHERE account_id = $1
	`, uuid.UUID(account))
}

func (s *Store) LinkOfUser(ctx context.Context, user discord.UserID) (links.Link, error) {
	return s.restored(ctx, `
		SELECT account_id, discord_user, flag, linked_at FROM links WHERE discord_user = $1
	`, string(user))
}

func (s *Store) restored(ctx context.Context, query string, arg any) (links.Link, error) {
	var (
		account  uuid.UUID
		user     string
		flag     sql.NullString
		linkedAt time.Time
	)
	err := s.db.On(ctx).QueryRowContext(ctx, query, arg).Scan(&account, &user, &flag, &linkedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return links.Link{}, links.ErrNoLink
	}
	if err != nil {
		return links.Link{}, fmt.Errorf("failed to read a link: %w", err)
	}
	return links.LinkSnapshot{
		Account: links.AccountID(account), User: discord.UserID(user), Flag: nations.Flag(flag.String),
		LinkedAt: linkedAt.UTC(),
	}.Restore(), nil
}

// The forgotten ones first: a user moves to another account in one call.
func (s *Store) SaveLinks(ctx context.Context, saved ...links.Link) error {
	return s.db.InTx(ctx, func(ctx context.Context) error {
		tx := s.db.On(ctx)
		for _, link := range saved {
			if kept := link.TakeSnapshot(); kept.Forgotten {
				if _, err := tx.ExecContext(ctx, `DELETE FROM links WHERE account_id = $1`, uuid.UUID(kept.Account)); err != nil {
					return fmt.Errorf("failed to delete a forgotten link: %w", err)
				}
			}
		}
		for _, link := range saved {
			kept := link.TakeSnapshot()
			if kept.Forgotten {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO links (account_id, discord_user, flag, linked_at) VALUES ($1, $2, $3, $4)
				ON CONFLICT (account_id) DO UPDATE SET
					discord_user = EXCLUDED.discord_user, flag = EXCLUDED.flag, linked_at = EXCLUDED.linked_at
			`, uuid.UUID(kept.Account), string(kept.User), sql.NullString{String: string(kept.Flag), Valid: kept.Flag != ""},
				kept.LinkedAt.UTC()); err != nil {
				return fmt.Errorf("failed to upsert a link: %w", err)
			}
		}
		return nil
	})
}
