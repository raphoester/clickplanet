package react_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/react_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type fakeBoard struct {
	changes []messages.ReactionChange
	given   messages.Reactions
	err     error
}

func (f *fakeBoard) React(_ context.Context, change messages.ReactionChange) (messages.Reactions, error) {
	f.changes = append(f.changes, change)
	if f.err != nil {
		return messages.Reactions{}, f.err
	}
	f.given = f.given.Applied(change)
	return f.given, nil
}

type fakeAuthors struct {
	author messages.Author
	ips    []string
	err    error
}

func (f *fakeAuthors) Author(_ context.Context, _ messages.AccountID, ip string) (messages.Author, error) {
	f.ips = append(f.ips, ip)
	return f.author, f.err
}

var (
	ada   = cpsession.AccountID{15: 1}
	clown = messages.Reaction(2)
	now   = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
)

func react(board *fakeBoard, authors *fakeAuthors, in react_usecase.In) ([]messages.Count, error) {
	ctx := cpctx.AddIPToContext(context.Background(), "1.2.3.4")
	return react_usecase.New(board, authors, cptime.NewFixedClock(now)).Execute(ctx, in)
}

func TestAPlayerReactsAsItsAccountAndIsAnsweredWithItsOwn(t *testing.T) {
	board := &fakeBoard{}
	authors := &fakeAuthors{author: messages.Author{Username: "ada", Tag: "a1b2c3"}}

	counts, err := react(board, authors, react_usecase.In{Account: ada, MessageID: "message-1", Reaction: clown, On: true})

	require.NoError(t, err)
	assert.Equal(t, []messages.Count{{Reaction: clown, Count: 1, Mine: true}}, counts)
	assert.Equal(t, []messages.ReactionChange{{
		MessageID: "message-1",
		Reaction:  clown,
		Reactor:   messages.ReactorOf(ada, authors.author),
		On:        true,
		At:        now,
	}}, board.changes)
	assert.Equal(t, []string{"1.2.3.4"}, authors.ips, "a guest is the tag of the address the reaction came from")
}

func TestAGuestReactsAsItsTag(t *testing.T) {
	board := &fakeBoard{}

	_, err := react(board, &fakeAuthors{author: messages.Author{Tag: "a1b2c3"}},
		react_usecase.In{Account: cpsession.NoAccount, MessageID: "message-1", Reaction: clown, On: true})

	require.NoError(t, err)
	assert.Equal(t, messages.Reactor("guest:a1b2c3"), board.changes[0].Reactor)
}

func TestAReactorThePlayerModuleCannotNameIsRefused(t *testing.T) {
	board := &fakeBoard{}

	_, err := react(board, &fakeAuthors{err: assert.AnError},
		react_usecase.In{Account: ada, MessageID: "message-1", Reaction: clown, On: true})

	require.ErrorIs(t, err, messages.ErrAuthorUnavailable)
	assert.Empty(t, board.changes)
}

func TestAnUnknownMessageIsReported(t *testing.T) {
	_, err := react(&fakeBoard{err: messages.ErrUnknownMessage}, &fakeAuthors{author: messages.Author{Tag: "a1b2c3"}},
		react_usecase.In{MessageID: "message-1", Reaction: clown, On: true})

	require.ErrorIs(t, err, messages.ErrUnknownMessage)
}
