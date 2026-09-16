// Package chatban puts a ban on the wire; both admin procedures that answer one send it.
package chatban

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
)

func Encode(ban bans.Ban) *chatv1.Ban {
	return &chatv1.Ban{
		AuthorTag: ban.AuthorTag,
		BannedAt:  timestamppb.New(ban.BannedAt),
		Reason:    ban.Reason,
	}
}
