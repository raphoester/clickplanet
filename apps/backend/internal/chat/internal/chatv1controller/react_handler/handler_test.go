package react_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/react_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/react_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type stubUseCase struct {
	ins    []react_usecase.In
	counts []messages.Count
	err    error
}

func (s *stubUseCase) Execute(_ context.Context, in react_usecase.In) ([]messages.Count, error) {
	s.ins = append(s.ins, in)
	return s.counts, s.err
}

func react(ctx context.Context, useCase *stubUseCase, reaction chatv1.Reaction) (*connect.Response[chatv1.ReactResponse], error) {
	req := connect.NewRequest(&chatv1.ReactRequest{MessageId: "message-1", Reaction: reaction, On: true})

	return react_handler.New(useCase).React(ctx, req) //nolint:wrapcheck // the test reads the handler's own error.
}

func TestReactMapsTheRequestAndTheAnswer(t *testing.T) {
	ada := cpsession.AccountID{15: 1}
	useCase := &stubUseCase{counts: []messages.Count{{Reaction: 2, Count: 4, Mine: true}}}

	res, err := react(cpctx.AddAccountToContext(t.Context(), ada.String()), useCase, chatv1.Reaction_REACTION_CLOWN)

	require.NoError(t, err)
	assert.Equal(t, []react_usecase.In{{
		Account:   ada,
		MessageID: "message-1",
		Reaction:  messages.Reaction(chatv1.Reaction_REACTION_CLOWN),
		On:        true,
	}}, useCase.ins)
	require.Len(t, res.Msg.GetReactions(), 1)
	assert.Equal(t, chatv1.Reaction_REACTION_CLOWN, res.Msg.GetReactions()[0].GetReaction())
	assert.Equal(t, uint32(4), res.Msg.GetReactions()[0].GetCount())
	assert.True(t, res.Msg.GetReactions()[0].GetMine())
}

func TestAReactionTheProtoDoesNotNameIsRefusedBeforeTheUseCase(t *testing.T) {
	for _, reaction := range []chatv1.Reaction{chatv1.Reaction_REACTION_UNSPECIFIED, chatv1.Reaction(9999)} {
		useCase := &stubUseCase{}

		_, err := react(t.Context(), useCase, reaction)

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		assert.Empty(t, useCase.ins)
	}
}

func TestReactMapsTheRefusals(t *testing.T) {
	for cause, code := range map[error]connect.Code{
		messages.ErrUnknownMessage:    connect.CodeNotFound,
		messages.ErrAuthorUnavailable: connect.CodeUnavailable,
	} {
		_, err := react(t.Context(), &stubUseCase{err: cause}, chatv1.Reaction_REACTION_CLOWN)

		assert.Equal(t, code, connect.CodeOf(err), cause.Error())
	}
}

func TestAnUnexpectedFailureIsLeftForTheErrorNet(t *testing.T) {
	_, err := react(t.Context(), &stubUseCase{err: assert.AnError}, chatv1.Reaction_REACTION_CLOWN)

	require.ErrorIs(t, err, assert.AnError)
}
