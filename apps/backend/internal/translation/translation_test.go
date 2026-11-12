package translation_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/translation"
)

func line(key, english string) translation.Line {
	return translation.Line{Key: translation.Key(key), English: english, Context: "where it shows"}
}

func TestAPlanSendsOnlyTheLinesThatAreNewOrWhoseEnglishChanged(t *testing.T) {
	source := []translation.Line{line("same", "Take a tile"), line("changed", "Catch a box now"), line("new", "Drop a bomb"), line("byHand", "Send")}
	current := translation.Catalogue{
		Lines: map[translation.Key]string{"same": "Prends une case", "changed": "Attrape une caisse", "byHand": "Envoyer", "gone": "Vieux"},
		Fingerprints: map[translation.Key]translation.Fingerprint{
			"same": translation.FingerprintOf("Take a tile"), "changed": translation.FingerprintOf("Catch a box"),
			"gone": translation.FingerprintOf("Old"),
		},
	}

	plan := translation.PlanOf("fr", source, current)

	assert.Equal(t, []translation.Line{line("changed", "Catch a box now"), line("new", "Drop a bomb")}, plan.Send)
	assert.Equal(t, 1, plan.Kept)
	assert.Equal(t, 1, plan.Adopted)
	assert.Equal(t, 1, plan.Dropped)
}

type answering struct {
	answers map[translation.Key]string
	asked   []translation.Line
	err     error
}

func (a *answering) Translate(_ context.Context, _ translation.Language, lines []translation.Line) (map[translation.Key]string, translation.Spend, error) {
	a.asked = append(a.asked, lines...)
	return a.answers, translation.Spend{InputTokens: 10, OutputTokens: 5}, a.err
}

func TestATranslationKeepsWhatWasKeptAndFingerprintsWhatCameBack(t *testing.T) {
	source := []translation.Line{line("kept", "Send"), line("tiles", "{count, plural, one {# tile} other {# tiles}}")}
	current := translation.Catalogue{
		Lines:        map[translation.Key]string{"kept": "Envoyer"},
		Fingerprints: map[translation.Key]translation.Fingerprint{"kept": translation.FingerprintOf("Send")},
	}
	translator := &answering{answers: map[translation.Key]string{"tiles": "{count, plural, one {# case} many {# de cases} other {# cases}}"}}

	outcome, err := translation.Translate(t.Context(), translation.PlanOf("fr", source, current), translator)

	require.NoError(t, err)
	assert.Equal(t, []translation.Line{line("tiles", "{count, plural, one {# tile} other {# tiles}}")}, translator.asked)
	assert.Equal(t, map[translation.Key]string{
		"kept": "Envoyer", "tiles": "{count, plural, one {# case} many {# de cases} other {# cases}}",
	}, outcome.Catalogue.Lines)
	assert.Equal(t, translation.FingerprintOf("{count, plural, one {# tile} other {# tiles}}"), outcome.Catalogue.Fingerprints["tiles"])
	assert.Equal(t, 1, outcome.Translated)
	assert.Equal(t, translation.Spend{InputTokens: 10, OutputTokens: 5}, outcome.Spend)
}

func TestATranslationThatLosesASlotIsNotWritten(t *testing.T) {
	source := []translation.Line{line("took", "{player} took {count} tiles"), line("silent", "Send")}
	translator := &answering{answers: map[translation.Key]string{"took": "{joueur} a pris {count} cases"}}

	outcome, err := translation.Translate(t.Context(), translation.PlanOf("fr", source, translation.EmptyCatalogue()), translator)

	require.NoError(t, err)
	assert.Empty(t, outcome.Catalogue.Lines)
	assert.Equal(t, []translation.Refusal{
		{Key: "took", Reason: "slots count:value joueur:value where the English has count:value player:value"},
		{Key: "silent", Reason: "no answer"},
	}, outcome.Refused)
}

func TestNothingToSendAsksNoTranslator(t *testing.T) {
	source := []translation.Line{line("kept", "Send")}
	current := translation.Catalogue{
		Lines:        map[translation.Key]string{"kept": "Envoyer"},
		Fingerprints: map[translation.Key]translation.Fingerprint{"kept": translation.FingerprintOf("Send")},
	}
	translator := &answering{err: errors.New("never asked")}

	outcome, err := translation.Translate(t.Context(), translation.PlanOf("fr", source, current), translator)

	require.NoError(t, err)
	assert.Empty(t, translator.asked)
	assert.Equal(t, "Envoyer", outcome.Catalogue.Lines["kept"])
}

func TestSlotsDiffer(t *testing.T) {
	for _, c := range []struct {
		name, english, translated, reason string
	}{
		{"same slots", "{player} took {count} tiles", "{count} cases prises par {player}", ""},
		{"a plural kept", "{count, plural, one {# tile} other {# tiles}}", "{count, plural, one {# pole} few {# pola} many {# pól} other {# pola}}", ""},
		{"a count as a plain number", "{count, plural, one {# tile} other {# tiles}}", "{count} petak", ""},
		{"words in the branches are not slots", "{goal, plural, one {Catch a box} other {Catch # boxes}}", "{goal, plural, one {Attrape une caisse} other {Attrape # caisses}}", ""},
		{"a tag kept", "Read the <terms>terms</terms>", "Lis les <terms>conditions</terms>", ""},
		{"a slot lost", "{winner} knocked out {loser}", "{winner} a éliminé", "slots winner:value where the English has loser:value winner:value"},
		{"a tag lost", "Read the <terms>terms</terms>", "Lis les conditions", "slots  where the English has terms:tag"},
		{"no other branch", "{count, plural, one {# tile} other {# tiles}}", "{count, plural, one {# case}}", `it does not read: a plural or a select has no "other" branch: {count}`},
		{"an open brace", "{count} tiles", "{count cases", "it does not read: an argument does not read: {count"},
	} {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.reason, translation.SlotsDiffer(c.english, c.translated))
		})
	}
}

func TestTheGlossaryGivesEachTermAWordInEachLanguage(t *testing.T) {
	glossary, err := translation.ReadGlossary()

	require.NoError(t, err)
	require.NotEmpty(t, glossary)
	for _, term := range glossary {
		assert.NotEmpty(t, term.Means, term.English)
		for _, language := range []translation.Language{"fr", "de", "tr", "pt-BR", "pl", "id", "es", "cs"} {
			assert.NotEmpty(t, term.Words[language], "%s in %s", term.English, language)
		}
	}
}
