package replyaddress

import (
	"connectrpc.com/connect"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
)

// Unset is empty: the address the account holds.
func Of(sent *string) (cpmail.Address, error) {
	if sent == nil {
		return "", nil
	}
	address, err := cpmail.AddressOf(*sent)
	if err != nil {
		return "", Refusal(connect.CodeInvalidArgument, cpmail.ErrAddressInvalid, supportv1.NoteRefusalReason_NOTE_REFUSAL_REASON_INVALID_REPLY_ADDRESS)
	}
	return address, nil
}

func Refusal(code connect.Code, sentinel error, reason supportv1.NoteRefusalReason) *connect.Error {
	refused := connect.NewError(code, sentinel)
	if detail, err := connect.NewErrorDetail(&supportv1.NoteRefusal{Reason: reason}); err == nil {
		refused.AddDetail(detail)
	}
	return refused
}
