package unban_member_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/unban_member_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/unban_member_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

type stubUseCase struct {
	in  unban_member_usecase.In
	out unban_member_usecase.Out
	err error
}

func (s *stubUseCase) Execute(
	_ context.Context,
	in unban_member_usecase.In,
) (unban_member_usecase.Out, error) {
	s.in = in
	return s.out, s.err
}

func unban(t *testing.T, useCase *stubUseCase) error {
	t.Helper()

	_, err := unban_member_handler.New(useCase).UnbanMember(t.Context(),
		connect.NewRequest(&chatv1.UnbanMemberRequest{AuthorTag: "a1b2c3"}))

	return err //nolint:wrapcheck // the code on the refusal is what is under test.
}

func TestUnbanMemberCarriesTheTagBack(t *testing.T) {
	useCase := &stubUseCase{out: unban_member_usecase.Out{AuthorTag: "a1b2c3"}}

	res, err := unban_member_handler.New(useCase).UnbanMember(t.Context(),
		connect.NewRequest(&chatv1.UnbanMemberRequest{AuthorTag: "#A1B2C3"}))

	require.NoError(t, err)
	assert.Equal(t, unban_member_usecase.In{AuthorTag: "#A1B2C3"}, useCase.in)
	assert.Equal(t, "a1b2c3", res.Msg.GetAuthorTag())
}

func TestUnbanMemberRefusesSomethingThatIsNotATag(t *testing.T) {
	err := unban(t, &stubUseCase{err: messages.ErrInvalidTag})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestUnbanMemberReportsAMemberNobodyBanned(t *testing.T) {
	err := unban(t, &stubUseCase{err: bans.ErrNotBanned})

	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestUnbanMemberLeavesAnythingElseToTheErrorNet(t *testing.T) {
	err := unban(t, &stubUseCase{err: assert.AnError})

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
