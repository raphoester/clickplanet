package moderation_query

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

const (
	DefaultLimit = 50
	MaxLimit     = 500
)

var (
	ErrUnknownAction = errors.New("an action the wire cannot carry")
	ErrUnknownKind   = errors.New("a kind of mute the wire cannot carry")
)

// Which actions: one account's, or everybody's for NoAccount, taken at Since or after.
type Filter struct {
	Account cpsession.AccountID
	Since   time.Time
	Limit   uint32
}

func NewPostgresQuery(db cppg.Conn) *PostgresQuery {
	return &PostgresQuery{db: db}
}

type PostgresQuery struct {
	db cppg.Conn
}

// $1 is the account, NULL for every account's, $2 the start, $3 the limit.
const actions = `
	SELECT id, account_id, action, acted_at, COALESCE(mute_kind::text, ''), COALESCE(seconds, 0),
		COALESCE(message_id, ''), COALESCE(text, ''), COALESCE(gif_url, ''), COALESCE(gif_width, 0),
		COALESCE(gif_height, 0), reason, operator_id, client
	FROM moderation
	WHERE ($1::uuid IS NULL OR account_id = $1) AND acted_at >= $2
	ORDER BY acted_at DESC, id DESC
	LIMIT $3
`

func (q *PostgresQuery) Actions(ctx context.Context, filter Filter) (*chatv1.ListModerationResponse, error) {
	rows, err := q.db.On(ctx).QueryContext(ctx, actions, accountOf(filter.Account), filter.Since.UTC(),
		limitOf(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("failed to read the moderation history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	answer := &chatv1.ListModerationResponse{Actions: []*chatv1.ModerationAction{}}
	for rows.Next() {
		var (
			id       int64
			account  uuid.NullUUID
			kind     string
			at       time.Time
			muteKind string
			seconds  int64
			message  string
			text     string
			gif      = &chatv1.Gif{}
			reason   string
			operator uuid.NullUUID
			client   string
		)
		if err := rows.Scan(&id, &account, &kind, &at, &muteKind, &seconds, &message, &text, &gif.Url, &gif.Width,
			&gif.Height, &reason, &operator, &client); err != nil {
			return nil, fmt.Errorf("failed to scan a moderation action: %w", err)
		}

		action := &chatv1.ModerationAction{
			ActionId: fmt.Sprint(id),
			ActedAt:  timestamppb.New(at),
			Reason:   reason,
			Client:   client,
		}
		if account.Valid {
			action.AccountId = account.UUID.String()
		}
		if operator.Valid {
			action.OperatorAccountId = operator.UUID.String()
		}
		if err := carry(action, kind, muteKind, seconds, message, text, gif); err != nil {
			return nil, fmt.Errorf("failed to read action %d: %w", id, err)
		}
		answer.Actions = append(answer.Actions, action)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the moderation history: %w", err)
	}
	return answer, nil
}

func carry(action *chatv1.ModerationAction, kind, muteKind string, seconds int64, message, text string, gif *chatv1.Gif) error {
	switch kind {
	case "mute":
		wire, err := muteKindOf(muteKind)
		if err != nil {
			return err
		}
		action.Action = &chatv1.ModerationAction_Mute{Mute: &chatv1.MuteAction{
			Kind: wire, Duration: durationpb.New(time.Duration(seconds) * time.Second),
		}}
	case "unmute":
		wire, err := muteKindOf(muteKind)
		if err != nil {
			return err
		}
		action.Action = &chatv1.ModerationAction_Unmute{Unmute: &chatv1.UnmuteAction{Kind: wire}}
	case "removal":
		removal := &chatv1.RemovalAction{MessageId: message}
		if gif.GetUrl() != "" {
			removal.Body = &chatv1.RemovalAction_Gif{Gif: gif}
		} else {
			removal.Body = &chatv1.RemovalAction_Text{Text: text}
		}
		action.Action = &chatv1.ModerationAction_Removal{Removal: removal}
	default:
		return fmt.Errorf("%w: %q", ErrUnknownAction, kind)
	}
	return nil
}

func muteKindOf(kept string) (chatv1.MuteKind, error) {
	switch kept {
	case "chat":
		return chatv1.MuteKind_MUTE_KIND_CHAT, nil
	case "gifs":
		return chatv1.MuteKind_MUTE_KIND_GIFS, nil
	default:
		return chatv1.MuteKind_MUTE_KIND_CHAT, fmt.Errorf("%w: %q", ErrUnknownKind, kept)
	}
}

func accountOf(account cpsession.AccountID) uuid.NullUUID {
	if account == cpsession.NoAccount {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: uuid.UUID(account), Valid: true}
}

func limitOf(asked uint32) uint32 {
	switch {
	case asked == 0:
		return DefaultLimit
	case asked > MaxLimit:
		return MaxLimit
	default:
		return asked
	}
}
