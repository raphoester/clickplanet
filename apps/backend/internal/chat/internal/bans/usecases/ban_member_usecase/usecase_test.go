package ban_member_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/usecases/ban_member_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type fakeBanner struct {
	banned []bans.Ban
	err    error
}

func (f *fakeBanner) Ban(_ context.Context, ban bans.Ban) (bans.Ban, error) {
	if f.err != nil {
		return bans.Ban{}, f.err
	}
	f.banned = append(f.banned, ban)
	return ban, nil
}

type fakeLog struct {
	redacted []string
	count    int
}

func (f *fakeLog) Redact(tag string) int {
	f.redacted = append(f.redacted, tag)
	return f.count
}

var noon = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

func newUseCase(banner *fakeBanner, log *fakeLog) *ban_member_usecase.UseCase {
	return ban_member_usecase.New(banner, log, cptime.NewFixedClock(noon))
}

func TestABanIsRecordedThenBroadcast(t *testing.T) {
	banner, log := &fakeBanner{}, &fakeLog{count: 3}

	out, err := newUseCase(banner, log).Execute(t.Context(), ban_member_usecase.In{
		AuthorTag: "#A1B2C3",
		Reason:    "slurs",
	})

	require.NoError(t, err)
	assert.Equal(t, []bans.Ban{{AuthorTag: "a1b2c3", BannedAt: noon, Reason: "slurs"}}, banner.banned)
	assert.Equal(t, []string{"a1b2c3"}, log.redacted)
	assert.Equal(t, 3, out.Redacted)
	assert.Equal(t, "a1b2c3", out.Ban.AuthorTag)
}

func TestSomethingThatIsNotATagBansNobody(t *testing.T) {
	banner, log := &fakeBanner{}, &fakeLog{}

	_, err := newUseCase(banner, log).Execute(t.Context(), ban_member_usecase.In{AuthorTag: "Bob"})

	require.ErrorIs(t, err, messages.ErrInvalidTag)
	assert.Empty(t, banner.banned)
	assert.Empty(t, log.redacted, "nothing is blanked for a ban that was never passed")
}

func TestABanThatCannotBeRecordedBlanksNothing(t *testing.T) {
	banner, log := &fakeBanner{err: assert.AnError}, &fakeLog{}

	_, err := newUseCase(banner, log).Execute(t.Context(), ban_member_usecase.In{AuthorTag: "a1b2c3"})

	require.ErrorIs(t, err, assert.AnError)
	assert.Empty(t, log.redacted)
}
