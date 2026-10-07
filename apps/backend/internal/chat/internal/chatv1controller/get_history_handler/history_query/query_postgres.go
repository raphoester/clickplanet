package history_query

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Authors interface {
	Authors(ctx context.Context, accounts []cpsession.AccountID) (map[cpsession.AccountID]*playerv1.Author, error)
}

const (
	DeletedName   = "[deleted]"
	NamedReactors = 20
)

var kinds = cpcolls.NewSet("bomb", "mute")

var (
	ErrUnknownReaction = errors.New("a kept reaction the proto does not name")
	ErrUnknownKind     = errors.New("a kept announcement of a kind nobody knows")
)

func NewPostgresQuery(db cppg.Querier, authors Authors, clock cptime.Clock, size int, retention time.Duration) *PostgresQuery {
	return &PostgresQuery{db: db, authors: authors, clock: clock, size: size, retention: retention}
}

type PostgresQuery struct {
	db        cppg.Querier
	authors   Authors
	clock     cptime.Clock
	size      int
	retention time.Duration
}

func (q *PostgresQuery) History(ctx context.Context, viewer cpsession.AccountID) (*chatv1.GetHistoryResponse, error) {
	since := q.clock.Now().Add(-q.retention)

	var (
		shown     []*chatv1.ChatMessage
		announced []*chatv1.Announcement
		seenUntil int64
	)
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var err error
		shown, err = q.shown(ctx, since, viewer)
		return err
	})
	group.Go(func() error {
		var err error
		announced, err = q.announced(ctx, since)
		return err
	})
	group.Go(func() error {
		var err error
		seenUntil, err = q.seenUntil(ctx, viewer)
		return err
	})
	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // each part already names what failed.
	}

	return &chatv1.GetHistoryResponse{Messages: shown, Announcements: announced, SeenUntilUnixMs: seenUntil}, nil
}

const seenMark = `
	SELECT floor(extract(epoch FROM seen_until) * 1000)::bigint
	FROM seen
	WHERE account_id = $1
`

func (q *PostgresQuery) seenUntil(ctx context.Context, viewer cpsession.AccountID) (int64, error) {
	if viewer == cpsession.NoAccount {
		return 0, nil
	}

	var until int64
	err := q.db.QueryRowContext(ctx, seenMark, uuid.UUID(viewer)).Scan(&until)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read until when the caller saw the chat: %w", err)
	}
	return until, nil
}

const inWindow = `
	SELECT seq, id, sent_at, account_id, name, author_admin, country, text
	FROM messages
	WHERE sent_at >= $1
	ORDER BY seq DESC
	LIMIT $2
`

const shownMessages = `
	WITH shown AS (` + inWindow + `)
	SELECT
		shown.id,
		floor(extract(epoch FROM shown.sent_at) * 1000)::bigint,
		shown.account_id,
		shown.name,
		shown.author_admin,
		shown.country,
		shown.text,
		COALESCE(versions.version, 0),
		COALESCE(tallies.counts, '[]')
	FROM shown
	LEFT JOIN reaction_versions versions ON versions.message_id = shown.id
	LEFT JOIN LATERAL (
		SELECT json_agg(json_build_object(
			'reaction', given.reaction,
			'count', given.count,
			'mine', given.mine,
			'accounts', given.accounts
		) ORDER BY given.first_at, given.reaction) AS counts
		FROM (
			SELECT
				reaction,
				count(*) AS count,
				COALESCE(bool_or(reactor = 'account:' || $3::uuid), false) AS mine,
				min(reacted_at) AS first_at,
				COALESCE(
					array_agg(substr(reactor, 9)::uuid ORDER BY reacted_at, reactor) FILTER (WHERE reactor LIKE 'account:%'),
					'{}'
				) AS accounts
			FROM reactions
			WHERE reactions.message_id = versions.message_id
			GROUP BY reaction
		) given
	) tallies ON true
	ORDER BY shown.seq
`

type row struct {
	id      string
	sentAt  int64
	account cpsession.AccountID
	name    string
	admin   bool
	country string
	text    string
	version uint64
	counts  []count
}

type count struct {
	Reaction int32       `json:"reaction"`
	Count    uint32      `json:"count"`
	Mine     bool        `json:"mine"`
	Accounts []uuid.UUID `json:"accounts"`
}

func (q *PostgresQuery) shown(
	ctx context.Context,
	since time.Time,
	viewer cpsession.AccountID,
) ([]*chatv1.ChatMessage, error) {
	rows, err := q.rows(ctx, since, viewer)
	if err != nil {
		return nil, err
	}

	named, err := q.authors.Authors(ctx, everyone(rows))
	if err != nil {
		return nil, fmt.Errorf("failed to read who the chat history is from: %w", err)
	}

	shown := make([]*chatv1.ChatMessage, 0, len(rows))
	for _, row := range rows {
		shown = append(shown, messageOf(row, named))
	}
	return shown, nil
}

func (q *PostgresQuery) rows(ctx context.Context, since time.Time, viewer cpsession.AccountID) ([]row, error) {
	result, err := q.db.QueryContext(ctx, shownMessages, since, q.size, nullable(viewer))
	if err != nil {
		return nil, fmt.Errorf("failed to read the chat history: %w", err)
	}
	defer func() { _ = result.Close() }()

	var rows []row
	for result.Next() {
		var (
			each    row
			account uuid.NullUUID
			counts  []byte
		)
		if err := result.Scan(
			&each.id, &each.sentAt, &account, &each.name, &each.admin, &each.country, &each.text,
			&each.version, &counts,
		); err != nil {
			return nil, fmt.Errorf("failed to scan a message of the chat history: %w", err)
		}
		if account.Valid {
			each.account = cpsession.AccountID(account.UUID)
		}
		if each.counts, err = countsOf(counts); err != nil {
			return nil, fmt.Errorf("failed to read the reactions to message %q: %w", each.id, err)
		}
		rows = append(rows, each)
	}
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the chat history: %w", err)
	}
	return rows, nil
}

func nullable(account cpsession.AccountID) uuid.NullUUID {
	return uuid.NullUUID{UUID: uuid.UUID(account), Valid: account != cpsession.NoAccount}
}

func countsOf(encoded []byte) ([]count, error) {
	var counts []count
	if err := json.Unmarshal(encoded, &counts); err != nil {
		return nil, fmt.Errorf("failed to decode the reactions: %w", err)
	}
	for _, each := range counts {
		if _, named := chatv1.Reaction_name[each.Reaction]; !named || each.Reaction == int32(chatv1.Reaction_REACTION_UNSPECIFIED) {
			return nil, fmt.Errorf("%w: %d", ErrUnknownReaction, each.Reaction)
		}
	}
	return counts, nil
}

func everyone(rows []row) []cpsession.AccountID {
	seen := cpcolls.NewSet[cpsession.AccountID]()
	accounts := make([]cpsession.AccountID, 0, len(rows))
	add := func(account cpsession.AccountID) {
		if account == cpsession.NoAccount || seen.Contains(account) {
			return
		}
		seen.Add(account)
		accounts = append(accounts, account)
	}

	for _, row := range rows {
		add(row.account)
	}
	for _, row := range rows {
		for _, each := range row.counts {
			for _, account := range each.Accounts {
				add(cpsession.AccountID(account))
			}
		}
	}
	return accounts
}

func messageOf(row row, named map[cpsession.AccountID]*playerv1.Author) *chatv1.ChatMessage {
	message := &chatv1.ChatMessage{
		Id:               row.id,
		SentAtUnixMs:     row.sentAt,
		CountryId:        row.country,
		Text:             row.text,
		Reactions:        reactionsOf(row.counts, named),
		ReactionsVersion: row.version,
	}

	if row.account == cpsession.NoAccount {
		message.AuthorName = row.name
		message.AuthorAdmin = row.admin
		return message
	}

	author, known := named[row.account]
	if !known {
		message.AuthorName = DeletedName
		return message
	}

	message.AuthorName = author.GetName()
	message.AuthorAdmin = author.GetAdmin()
	message.AuthorColor = author.GetColor()
	message.AuthorStreak = author.GetStreak()
	message.AuthorTitle = author.GetWornTitle()
	return message
}

func reactionsOf(counts []count, named map[cpsession.AccountID]*playerv1.Author) []*chatv1.ReactionCount {
	encoded := make([]*chatv1.ReactionCount, 0, len(counts))
	for _, each := range counts {
		encoded = append(encoded, &chatv1.ReactionCount{
			Reaction: chatv1.Reaction(each.Reaction),
			Count:    each.Count,
			Mine:     each.Mine,
			Reactors: namesOf(each.Accounts, named),
		})
	}
	return encoded
}

func namesOf(accounts []uuid.UUID, named map[cpsession.AccountID]*playerv1.Author) []string {
	names := make([]string, 0, min(len(accounts), NamedReactors))
	for _, account := range accounts {
		if len(names) == NamedReactors {
			break
		}
		if author, known := named[cpsession.AccountID(account)]; known {
			names = append(names, author.GetName())
		}
	}
	return names
}

const shownAnnouncements = `
	WITH shown AS (` + inWindow + `),
	beginning AS (
		SELECT CASE
			WHEN count(*) < $2 THEN $1::timestamptz
			ELSE (array_agg(sent_at ORDER BY seq))[1]
		END AS at
		FROM shown
	)
	SELECT
		newest.id::text,
		floor(extract(epoch FROM newest.announced_at) * 1000)::bigint,
		newest.kind,
		newest.payload::text
	FROM (
		SELECT id, kind, payload, announced_at
		FROM announcements
		WHERE announced_at >= (SELECT at FROM beginning)
		ORDER BY announced_at DESC, id DESC
		LIMIT $2
	) newest
	ORDER BY newest.announced_at, newest.id
`

func (q *PostgresQuery) announced(ctx context.Context, since time.Time) ([]*chatv1.Announcement, error) {
	result, err := q.db.QueryContext(ctx, shownAnnouncements, since, q.size)
	if err != nil {
		return nil, fmt.Errorf("failed to read the chat announcements: %w", err)
	}
	defer func() { _ = result.Close() }()

	announced := make([]*chatv1.Announcement, 0)
	for result.Next() {
		announcement := &chatv1.Announcement{}
		if err := result.Scan(
			&announcement.Id, &announcement.AnnouncedAtUnixMs, &announcement.Kind, &announcement.Payload,
		); err != nil {
			return nil, fmt.Errorf("failed to scan a chat announcement: %w", err)
		}
		if !kinds.Contains(announcement.GetKind()) {
			return nil, fmt.Errorf("%w: %q", ErrUnknownKind, announcement.GetKind())
		}
		announced = append(announced, announcement)
	}
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the chat announcements: %w", err)
	}
	return announced, nil
}
