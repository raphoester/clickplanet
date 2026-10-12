package notes_test

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var phone = notes.Diagnostics{
	Build: "play-a1b2c3.js", ViewportWidth: 390, ViewportHeight: 844, PixelRatio: 3, ConnectionDown: true,
	UserAgent: "Mozilla/5.0 (iPhone)", GPUVendor: "Apple Inc.", GPURenderer: "Apple GPU",
	WebGLVersion: "WebGL 2.0", MaxTextureSize: 4096, ContextLost: true, Cores: 6, MemoryGB: 4,
}

func diagnosticsKept(t *testing.T, said notes.Diagnostics) notes.Diagnostics {
	t.Helper()

	note, err := notes.NewBugReport(notes.NoteID{15: 1}, ada, "the globe is black", said, sentAt)
	require.NoError(t, err)
	return note.TakeSnapshot().Diagnostics
}

func TestEveryKindOfNoteKeepsWhatThePageSaid(t *testing.T) {
	bug, err := notes.NewBugReport(notes.NoteID{15: 1}, ada, "the globe is black", phone, sentAt)
	require.NoError(t, err)
	suggestion, err := notes.NewSuggestion(notes.NoteID{15: 2}, ada, "a night mode", phone, sentAt)
	require.NoError(t, err)

	assert.Equal(t, phone, bug.TakeSnapshot().Diagnostics)
	assert.Equal(t, phone, suggestion.TakeSnapshot().Diagnostics)
}

func TestANoteKeepsItsDiagnosticsCutAndCleaned(t *testing.T) {
	kept := diagnosticsKept(t, notes.Diagnostics{
		Build:          " play\x00-\n" + strings.Repeat("a", 100),
		ViewportWidth:  4_000_000_000,
		ViewportHeight: 4_000_000_000,
		PixelRatio:     2.6249,
		UserAgent:      strings.Repeat("🌍", 300) + string([]byte{0xff}),
		GPUVendor:      " Apple\tInc. ",
		GPURenderer:    strings.Repeat("g", 300),
		WebGLVersion:   strings.Repeat("w", 300),
		MaxTextureSize: 4_000_000_000,
		Cores:          4_000_000_000,
		MemoryGB:       1e9,
	})

	assert.Equal(t, "play-"+strings.Repeat("a", 59), kept.Build, "64 characters, with no control character")
	assert.Equal(t, uint32(100_000), kept.ViewportWidth)
	assert.Equal(t, uint32(100_000), kept.ViewportHeight)
	assert.InDelta(t, 2.62, kept.PixelRatio, 1e-9, "two decimals")
	assert.Equal(t, 256, utf8.RuneCountInString(kept.UserAgent))
	assert.True(t, utf8.ValidString(kept.UserAgent))
	assert.Equal(t, "AppleInc.", kept.GPUVendor)
	assert.Len(t, kept.GPURenderer, 256)
	assert.Len(t, kept.WebGLVersion, 128)
	assert.Equal(t, uint32(1<<20), kept.MaxTextureSize)
	assert.Equal(t, uint32(1024), kept.Cores)
	assert.InDelta(t, 1024, kept.MemoryGB, 1e-9)
}

func TestAMeasureThatIsNotOneIsKeptAsNone(t *testing.T) {
	for name, value := range map[string]float64{"not a number": math.NaN(), "negative": -2, "minus infinity": math.Inf(-1)} {
		kept := diagnosticsKept(t, notes.Diagnostics{PixelRatio: value, MemoryGB: value})

		assert.Zero(t, kept.PixelRatio, name)
		assert.Zero(t, kept.MemoryGB, name)
	}

	kept := diagnosticsKept(t, notes.Diagnostics{PixelRatio: math.Inf(1), MemoryGB: math.Inf(1)})
	assert.InDelta(t, 16, kept.PixelRatio, 1e-9, "the most a pixel ratio is kept as")
	assert.InDelta(t, 1024, kept.MemoryGB, 1e-9)
}

func TestANoteWhosePageSaidNothingKeepsEmptyDiagnostics(t *testing.T) {
	assert.Equal(t, notes.Diagnostics{}, diagnosticsKept(t, notes.Diagnostics{Build: " \n", UserAgent: "\x00"}))
}

func TestABugReportIsANoteOfThatKindWithItsTextCleaned(t *testing.T) {
	note, err := notes.NewBugReport(notes.NoteID{15: 1}, ada, "  the globe is black  ", notes.Diagnostics{}, sentAt)

	require.NoError(t, err)
	assert.Equal(t, notes.Snapshot{
		ID: notes.NoteID{15: 1}, Kind: notes.KindBug, Account: ada, Text: "the globe is black", At: sentAt,
	}, note.TakeSnapshot())
}

func TestABugReportNeedsTheAccountThatSendsIt(t *testing.T) {
	_, err := notes.NewBugReport(notes.NoteID{15: 1}, notes.AccountID{}, "the globe is black", notes.Diagnostics{}, sentAt)

	require.ErrorIs(t, err, notes.ErrInvalidNote)
}

func TestAnAnonymizedNoteKeepsItsDiagnostics(t *testing.T) {
	note, err := notes.NewBugReport(notes.NoteID{15: 1}, ada, "the globe is black", phone, sentAt)
	require.NoError(t, err)

	anonymized := note.Anonymized().TakeSnapshot()

	assert.Equal(t, notes.AccountID{}, anonymized.Account)
	assert.Equal(t, phone, anonymized.Diagnostics)
}
