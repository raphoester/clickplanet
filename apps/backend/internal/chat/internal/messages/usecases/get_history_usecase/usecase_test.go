package get_history_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/usecases/get_history_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/inmemory_reaction_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	expired = announcements.AnnouncementID{15: 1}
	old     = announcements.AnnouncementID{15: 2}
	middle  = announcements.AnnouncementID{15: 3}
	latest  = announcements.AnnouncementID{15: 4}
)

var (
	ada   = cpsession.AccountID{15: 1}
	now   = time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	clown = reactions.Reaction(2)
	other = cpsession.AccountID{15: 2}
)

// fakeAuthors is the player module, which knows what each account is called. An account it is not told about
// is one nobody can name any more: deleted.
type fakeAuthors struct {
	named map[messages.AccountID]messages.Author
	asked [][]messages.AccountID
	err   error
}

func (f *fakeAuthors) Authors(
	_ context.Context,
	accounts []messages.AccountID,
) (map[messages.AccountID]messages.Author, error) {
	f.asked = append(f.asked, accounts)
	if f.err != nil {
		return nil, f.err
	}

	found := make(map[messages.AccountID]messages.Author, len(accounts))
	for _, account := range accounts {
		if author, known := f.named[account]; known {
			found[account] = author
		}
	}
	return found, nil
}

type fixture struct {
	messages      *inmemory_message_storage.Storage
	reactions     *inmemory_reaction_storage.Storage
	announcements *inmemory_announcement_storage.Storage
	authors       *fakeAuthors
}

func newFixture(t *testing.T, texts ...string) fixture {
	t.Helper()

	f := fixture{
		messages:      inmemory_message_storage.New(),
		reactions:     inmemory_reaction_storage.New(),
		announcements: inmemory_announcement_storage.New(),
		authors: &fakeAuthors{named: map[messages.AccountID]messages.Author{
			ada:   {Name: "Ada"},
			other: {Name: "Bob", Admin: true},
		}},
	}
	for i, text := range texts {
		f.sent(t, messages.Message{
			ID: messages.MessageID(text), Account: ada, Text: text,
			SentAt: now.Add(time.Duration(i-len(texts)) * time.Hour),
		})
	}
	return f
}

func (f fixture) sent(t *testing.T, message messages.Message) {
	t.Helper()

	require.NoError(t, f.messages.Append(t.Context(), messages.Record{Message: message}))
}

func (f fixture) read(t *testing.T, account messages.AccountID) get_history_usecase.History {
	t.Helper()

	useCase := get_history_usecase.New(f.messages, f.reactions, f.announcements, f.authors,
		cptime.NewFixedClock(now), messages.Window{Size: 2, Retention: 24 * time.Hour})
	history, err := useCase.Execute(t.Context(), account)
	require.NoError(t, err)
	return history
}

func (f fixture) history(t *testing.T, account messages.AccountID) []get_history_usecase.Entry {
	t.Helper()

	return f.read(t, account).Messages
}

func texts(history []get_history_usecase.Entry) []string {
	texts := make([]string, 0, len(history))
	for _, entry := range history {
		texts = append(texts, entry.Message.Text)
	}
	return texts
}

func TestTheHistoryIsTheWindowOfNewestMessages(t *testing.T) {
	f := newFixture(t, "old", "middle", "new")

	assert.Equal(t, []string{"middle", "new"}, texts(f.history(t, ada)))
}

func TestEachMessageCarriesItsReactionsMarkedForTheCallersAccount(t *testing.T) {
	f := newFixture(t, "hello")
	require.NoError(t, f.reactions.Save(t.Context(), reactions.Change{
		MessageID: "hello", Reaction: clown, Reactor: reactions.ReactorOf(ada), On: true, At: now,
	}))

	assert.Equal(t, []reactions.Count{{Reaction: clown, Count: 1, Mine: true}}, f.history(t, ada)[0].Reactions)
	assert.Equal(t, uint64(1), f.history(t, ada)[0].ReactionsVersion)
	assert.Equal(t, []reactions.Count{{Reaction: clown, Count: 1, Mine: false}},
		f.history(t, other)[0].Reactions, "another account")
	assert.Equal(t, []reactions.Count{{Reaction: clown, Count: 1, Mine: false}},
		f.history(t, cpsession.NoAccount)[0].Reactions, "no token")
}

func TestTheHistoryCarriesTheWindowOfNewestAnnouncements(t *testing.T) {
	f := newFixture(t, "hello")
	ago := map[announcements.AnnouncementID]time.Duration{expired: 40, old: 20, middle: 10, latest: 4}
	for _, id := range []announcements.AnnouncementID{expired, old, middle, latest} {
		require.NoError(t, f.announcements.Append(t.Context(), announcements.Announcement{
			ID: id, Kind: announcements.KindBomb, At: now.Add(-ago[id] * time.Hour),
		}))
	}

	history := f.read(t, ada)

	ids := make([]announcements.AnnouncementID, 0, len(history.Announcements))
	for _, announcement := range history.Announcements {
		ids = append(ids, announcement.ID)
	}
	assert.Equal(t, []announcements.AnnouncementID{middle, latest}, ids)
	assert.Equal(t, []string{"hello"}, texts(history.Messages), "announcements do not take the messages' places")
}

func TestEachMessageIsNamedByWhoItsAccountIsNow(t *testing.T) {
	f := newFixture(t, "hello")

	shown := f.history(t, ada)[0].Message

	assert.Equal(t, "Ada", shown.AuthorName, "not a copy kept when it was sent")
	assert.False(t, shown.AuthorAdmin)
	assert.Equal(t, [][]messages.AccountID{{ada}}, f.authors.asked)
}

func TestARenameShowsOnEverythingItsPlayerEverSaid(t *testing.T) {
	f := newFixture(t, "one", "two")
	f.authors.named[ada] = messages.Author{Name: "Ada Lovelace", Admin: true}

	for _, entry := range f.history(t, ada) {
		assert.Equal(t, "Ada Lovelace", entry.Message.AuthorName, entry.Message.Text)
		assert.True(t, entry.Message.AuthorAdmin, entry.Message.Text)
	}
}

func TestOneAccountIsAskedAboutOnceHoweverMuchItSaid(t *testing.T) {
	f := newFixture(t, "one", "two")

	f.history(t, ada)

	assert.Equal(t, [][]messages.AccountID{{ada}}, f.authors.asked, "a page costs one ask, not one per message")
}

func TestADeletedAccountIsNoLongerNamed(t *testing.T) {
	f := newFixture(t, "hello")
	delete(f.authors.named, ada)

	shown := f.history(t, ada)[0].Message

	assert.Equal(t, messages.DeletedName, shown.AuthorName, "showing the old name would undo the deletion")
	assert.False(t, shown.AuthorAdmin)
	assert.Equal(t, "hello", shown.Text, "what it said stays: a thread keeps its shape")
}

func TestAMessageFromBeforeAccountsKeepsTheNameItCarries(t *testing.T) {
	f := newFixture(t)
	f.sent(t, messages.Message{
		ID: "legacy", Text: "legacy", SentAt: now.Add(-time.Hour),
		AuthorName: "Bob", AuthorAdmin: true,
	})

	shown := f.history(t, ada)[0].Message

	assert.Equal(t, "Bob", shown.AuthorName, "it has no account to read a name from")
	assert.True(t, shown.AuthorAdmin)
	assert.Empty(t, f.authors.asked[0], "and nobody is asked about it")
}

func TestAHistoryNobodyCanBeNamedInIsARefusal(t *testing.T) {
	f := newFixture(t, "hello")
	f.authors.err = errors.New("the player module is down")

	useCase := get_history_usecase.New(f.messages, f.reactions, f.announcements, f.authors,
		cptime.NewFixedClock(now), messages.Window{Size: 2, Retention: 24 * time.Hour})
	_, err := useCase.Execute(t.Context(), ada)

	assert.Error(t, err, "a chat of anonymous messages is worse than none")
}
