package chatv1controller

import (
	"errors"

	"connectrpc.com/connect"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1/chatv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/connectutil"
)

type MessageLimiter = connectutil.Limiter

type SenderBlocklist = connectutil.Blocklist

var (
	ErrTooManyMessages = errors.New("too many messages")
	ErrSenderBlocked   = errors.New("message refused")
)

// NewRateLimitInterceptor throttles SendMessage only. GetHistory is a single
// read on join, and limiting it would punish a page load rather than a bot.
//
// SendMessage is a public, unauthenticated write that fans out to every
// connected client, so this runs at the edge — ahead of decoding the message
// and well ahead of validating it, which makes a flood of malformed messages
// cost a sender exactly what a flood of well-formed ones does.
func NewRateLimitInterceptor(limiter MessageLimiter) connect.Interceptor {
	return connectutil.NewRateLimitInterceptor(
		limiter,
		ErrTooManyMessages,
		chatv1connect.ChatServiceSendMessageProcedure,
	)
}

// NewBlocklistInterceptor cuts a blocked sender off from the chat entirely,
// reads included: it is the config-driven way to stop an abusive address with a
// restart rather than a rebuild. Entries are prefixes, so a whole range goes in
// one line.
//
// It wraps the limiter rather than sitting inside it, so a refused sender does
// not also spend a token — otherwise their next message would come back 429 and
// say the wrong thing about why they were refused.
func NewBlocklistInterceptor(blocklist SenderBlocklist) connect.Interceptor {
	return connectutil.NewIPBlockInterceptor(
		blocklist,
		ErrSenderBlocked,
		nil, // no counter: this list is hand-maintained, so its size is already known
		chatv1connect.ChatServiceSendMessageProcedure,
		chatv1connect.ChatServiceGetHistoryProcedure,
	)
}
