package ban_member_handler_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/ban_member_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/ban_member_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type stubUseCase struct {
	in  ban_member_usecase.In
	out ban_member_usecase.Out
	err error
}

func (s *stubUseCase) Execute(_ context.Context, in ban_member_usecase.In) (ban_member_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

var noon = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

func TestBanMemberCarriesTheBanBack(t *testing.T) {
	useCase := &stubUseCase{out: ban_member_usecase.Out{
		Ban:      bans.Ban{AuthorTag: "a1b2c3", BannedAt: noon, Reason: "slurs"},
		Redacted: 4,
	}}

	res, err := ban_member_handler.New(useCase).BanMember(t.Context(),
		connect.NewRequest(&chatv1.BanMemberRequest{AuthorTag: "#a1b2c3", Reason: "slurs"}))

	require.NoError(t, err)
	assert.Equal(t, ban_member_usecase.In{AuthorTag: "#a1b2c3", Reason: "slurs"}, useCase.in)
	assert.Equal(t, "a1b2c3", res.Msg.GetBan().GetAuthorTag())
	assert.Equal(t, "slurs", res.Msg.GetBan().GetReason())
	assert.Equal(t, noon, res.Msg.GetBan().GetBannedAt().AsTime())
	assert.Equal(t, uint32(4), res.Msg.GetRedacted())
}

func TestBanMemberRefusesSomethingThatIsNotATag(t *testing.T) {
	useCase := &stubUseCase{err: messages.ErrInvalidTag}

	_, err := ban_member_handler.New(useCase).BanMember(t.Context(),
		connect.NewRequest(&chatv1.BanMemberRequest{AuthorTag: "Bob"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestBanMemberLeavesAnythingElseToTheErrorNet(t *testing.T) {
	useCase := &stubUseCase{err: assert.AnError}

	_, err := ban_member_handler.New(useCase).BanMember(t.Context(),
		connect.NewRequest(&chatv1.BanMemberRequest{AuthorTag: "a1b2c3"}))

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
