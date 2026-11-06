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
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const NamedReactors = 20

var kinds = cpcolls.NewSet("bomb", "mute", "round", "fortify", "champion", "flag_mission")

var (
	ErrUnknownReaction = errors.New("a kept reaction the proto does not name")
	ErrUnknownKind     = errors.New("a kept announcement of a kind nobody knows")
)

func NewPostgresQuery(db cppg.Conn, clock cptime.Clock, size int, retention time.Duration) *PostgresQuery {
	return &PostgresQuery{db: db, clock: clock, size: size, retention: retention}
}

type PostgresQuery struct {
	db        cppg.Conn
	clock     cptime.Clock
	size      int
	retention time.Duration
}

// A flag's room shows the announcements told in it, and the world every one.
func (q *PostgresQuery) History(
	ctx context.Context,
	viewer cpsession.AccountID,
	room messages.Room,
) (*chatv1.GetHistoryResponse, error) {
	since := q.clock.Now().Add(-q.retention)

	var (
		shown     []*chatv1.ChatMessage
		announced = make([]*chatv1.Announcement, 0)
		seenUntil int64
	)
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var err error
		shown, err = q.shown(ctx, since, room, viewer)
		return err
	})
	group.Go(func() error {
		var err error
		announced, err = q.announced(ctx, since, room)
		return err
	})
	group.Go(func() error {
		var err error
		seenUntil, err = q.seenUntil(ctx, viewer, room)
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
	WHERE account_id = $1 AND room = $2
`

func (q *PostgresQuery) seenUntil(ctx context.Context, viewer cpsession.AccountID, room messages.Room) (int64, error) {
	if viewer == cpsession.NoAccount {
		return 0, nil
	}

	var until int64
	err := q.db.On(ctx).QueryRowContext(ctx, seenMark, uuid.UUID(viewer), string(room)).Scan(&until)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to read until when the caller saw the chat: %w", err)
	}
	return until, nil
}

const inWindow = `
	SELECT seq, id, sent_at, account_id, country, text, reply_to,
		EXISTS (SELECT 1 FROM removals WHERE removals.message_id = messages.id) AS removed
	FROM messages
	WHERE room = $3 AND sent_at >= $1
	ORDER BY seq DESC
	LIMIT $2
`

const shownMessages = `
	WITH shown AS (` + inWindow + `)
	SELECT
		shown.id,
		floor(extract(epoch FROM shown.sent_at) * 1000)::bigint,
		shown.account_id,
		shown.country,
		CASE WHEN shown.removed THEN '' ELSE shown.text END,
		shown.removed,
		CASE WHEN shown.removed THEN '' ELSE COALESCE(gifs.url, '') END,
		COALESCE(gifs.width, 0),
		COALESCE(gifs.height, 0),
		COALESCE(quoted.id, ''),
		quoted.account_id,
		CASE WHEN quoted.removed THEN '' ELSE COALESCE(quoted.text, '') END,
		COALESCE(quoted.removed, false),
		CASE WHEN quoted.removed THEN '' ELSE COALESCE(quoted_gifs.url, '') END,
		COALESCE(quoted_gifs.width, 0),
		COALESCE(quoted_gifs.height, 0),
		COALESCE(versions.version, 0),
		COALESCE(tallies.counts, '[]')
	FROM shown
	LEFT JOIN gifs ON gifs.message_seq = shown.seq
	LEFT JOIN LATERAL (
		SELECT seq, id, account_id, text,
			EXISTS (SELECT 1 FROM removals WHERE removals.message_id = messages.id) AS removed
		FROM messages
		WHERE messages.id = shown.reply_to AND messages.sent_at >= $1
		ORDER BY seq DESC
		LIMIT 1
	) quoted ON true
	LEFT JOIN gifs quoted_gifs ON quoted_gifs.message_seq = quoted.seq
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
				COALESCE(bool_or(reactor = 'account:' || $4::uuid), false) AS mine,
				min(reacted_at) AS first_at,
				COALESCE(
					array_agg(substr(reactor, 9)::uuid ORDER BY reacted_at, reactor) FILTER (WHERE reactor LIKE 'account:%'),
					'{}'
				) AS accounts
			FROM reactions
			WHERE reactions.message_id = versions.message_id AND NOT shown.removed
			GROUP BY reaction
		) given
	) tallies ON true
	ORDER BY shown.seq
`

type row struct {
	id      string
	sentAt  int64
	account cpsession.AccountID
	country string
	text    string
	removed bool
	gif     gif
	quoted  quoted
	version uint64
	counts  []count
}

type quoted struct {
	id      string
	account cpsession.AccountID
	text    string
	removed bool
	gif     gif
}

type gif struct {
	url    string
	width  uint32
	height uint32
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
	room messages.Room,
	viewer cpsession.AccountID,
) ([]*chatv1.ChatMessage, error) {
	rows, err := q.rows(ctx, since, room, viewer)
	if err != nil {
		return nil, err
	}

	shown := make([]*chatv1.ChatMessage, 0, len(rows))
	for _, row := range rows {
		shown = append(shown, messageOf(row))
	}
	return shown, nil
}

func (q *PostgresQuery) rows(
	ctx context.Context,
	since time.Time,
	room messages.Room,
	viewer cpsession.AccountID,
) ([]row, error) {
	result, err := q.db.On(ctx).QueryContext(ctx, shownMessages, since, q.size, string(room), nullable(viewer))
	if err != nil {
		return nil, fmt.Errorf("failed to read the chat history: %w", err)
	}
	defer func() { _ = result.Close() }()

	var rows []row
	for result.Next() {
		var (
			each          row
			account       uuid.NullUUID
			quotedAccount uuid.NullUUID
			counts        []byte
		)
		if err := result.Scan(
			&each.id, &each.sentAt, &account, &each.country, &each.text, &each.removed,
			&each.gif.url, &each.gif.width, &each.gif.height,
			&each.quoted.id, &quotedAccount, &each.quoted.text, &each.quoted.removed,
			&each.quoted.gif.url, &each.quoted.gif.width, &each.quoted.gif.height,
			&each.version, &counts,
		); err != nil {
			return nil, fmt.Errorf("failed to scan a message of the chat history: %w", err)
		}
		if account.Valid {
			each.account = cpsession.AccountID(account.UUID)
		}
		if quotedAccount.Valid {
			each.quoted.account = cpsession.AccountID(quotedAccount.UUID)
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

func messageOf(row row) *chatv1.ChatMessage {
	message := &chatv1.ChatMessage{
		Id:               row.id,
		SentAtUnixMs:     row.sentAt,
		AuthorAccountId:  accountOf(row.account),
		CountryId:        row.country,
		Removed:          row.removed,
		Reactions:        reactionsOf(row.counts),
		ReactionsVersion: row.version,
		ReplyTo:          quoteOf(row.quoted),
	}
	if gif := gifOf(row.gif); gif != nil {
		message.Body = &chatv1.ChatMessage_Gif{Gif: gif}
	} else if !row.removed {
		message.Body = &chatv1.ChatMessage_Text{Text: row.text}
	}
	return message
}

func quoteOf(quoted quoted) *chatv1.Quote {
	if quoted.id == "" {
		return nil
	}

	quote := &chatv1.Quote{MessageId: quoted.id, AuthorAccountId: accountOf(quoted.account), Removed: quoted.removed}
	if gif := gifOf(quoted.gif); gif != nil {
		quote.Body = &chatv1.Quote_Gif{Gif: gif}
	} else if !quoted.removed {
		quote.Body = &chatv1.Quote_Text{Text: quoted.text}
	}
	return quote
}

func accountOf(account cpsession.AccountID) string {
	if account == cpsession.NoAccount {
		return ""
	}
	return account.String()
}

func gifOf(gif gif) *chatv1.Gif {
	if gif.url == "" {
		return nil
	}
	return &chatv1.Gif{Url: gif.url, Width: gif.width, Height: gif.height}
}

func reactionsOf(counts []count) []*chatv1.ReactionCount {
	encoded := make([]*chatv1.ReactionCount, 0, len(counts))
	for _, each := range counts {
		encoded = append(encoded, &chatv1.ReactionCount{
			Reaction:          chatv1.Reaction(each.Reaction),
			Count:             each.Count,
			Mine:              each.Mine,
			ReactorAccountIds: idsOf(each.Accounts),
		})
	}
	return encoded
}

func idsOf(accounts []uuid.UUID) []string {
	ids := make([]string, 0, min(len(accounts), NamedReactors))
	for _, account := range accounts[:min(len(accounts), NamedReactors)] {
		ids = append(ids, account.String())
	}
	return ids
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
		WHERE announced_at >= (SELECT at FROM beginning) AND ($3 = $4 OR room = $3)
		ORDER BY announced_at DESC, id DESC
		LIMIT $2
	) newest
	ORDER BY newest.announced_at, newest.id
`

func (q *PostgresQuery) announced(ctx context.Context, since time.Time, room messages.Room) ([]*chatv1.Announcement, error) {
	result, err := q.db.On(ctx).QueryContext(ctx, shownAnnouncements, since, q.size, string(room), string(messages.World))
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
