package excerpt_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-ops/internal/excerpt"
)

func everyLine([]byte) bool { return true }

func TestLinesUnderTheLimitAreAllKept(t *testing.T) {
	collector := excerpt.NewCollector(excerpt.Limit{Lines: 3, Bytes: 100})

	require.NoError(t, collector.Fill(strings.NewReader("one\ntwo\n"), everyLine))

	assert.Equal(t, excerpt.Excerpt{Lines: []string{"one", "two"}}, collector.Excerpt())
	assert.False(t, collector.Full())
}

func TestTheLineAfterTheLastAllowedOneCutsTheExcerpt(t *testing.T) {
	collector := excerpt.NewCollector(excerpt.Limit{Lines: 2, Bytes: 100})

	require.NoError(t, collector.Fill(strings.NewReader("one\ntwo\nthree\nfour\n"), everyLine))

	assert.Equal(t, excerpt.Excerpt{Lines: []string{"one", "two"}, Truncated: true}, collector.Excerpt())
	assert.True(t, collector.Full())
}

func TestALineThatWouldPassTheByteLimitCutsTheExcerpt(t *testing.T) {
	collector := excerpt.NewCollector(excerpt.Limit{Lines: 10, Bytes: 7})

	require.NoError(t, collector.Fill(strings.NewReader("one\ntwo\nthree\n"), everyLine))

	assert.Equal(t, excerpt.Excerpt{Lines: []string{"one", "two"}, Truncated: true}, collector.Excerpt())
}

func TestOnlyTheLinesAskedForAreKept(t *testing.T) {
	collector := excerpt.NewCollector(excerpt.Limit{Lines: 10, Bytes: 100})
	withAnO := func(line []byte) bool { return bytes.Contains(line, []byte("o")) }

	require.NoError(t, collector.Fill(strings.NewReader("one\ntwo\nthree\n"), withAnO))

	assert.Equal(t, []string{"one", "two"}, collector.Excerpt().Lines)
}

func TestALineLeftOutDoesNotCountTowardsTheLimit(t *testing.T) {
	collector := excerpt.NewCollector(excerpt.Limit{Lines: 1, Bytes: 100})
	onlyThree := func(line []byte) bool { return string(line) == "three" }

	require.NoError(t, collector.Fill(strings.NewReader("one\ntwo\nthree\n"), onlyThree))

	assert.Equal(t, excerpt.Excerpt{Lines: []string{"three"}}, collector.Excerpt())
}

func TestSeveralSourcesFillOneExcerpt(t *testing.T) {
	collector := excerpt.NewCollector(excerpt.Limit{Lines: 3, Bytes: 100})

	require.NoError(t, collector.Fill(strings.NewReader("one\ntwo\n"), everyLine))
	require.NoError(t, collector.Fill(strings.NewReader("three\nfour\n"), everyLine))

	assert.Equal(t, excerpt.Excerpt{Lines: []string{"one", "two", "three"}, Truncated: true}, collector.Excerpt())
}
