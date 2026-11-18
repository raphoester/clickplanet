package translation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

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

	plan := translation.PlanOf("fr", source, current, translation.NoMemory{})

	assert.Equal(t, []translation.Line{line("changed", "Catch a box now"), line("new", "Drop a bomb")}, plan.Send)
	assert.Equal(t, 1, plan.Kept)
	assert.Equal(t, 1, plan.Adopted)
	assert.Equal(t, 1, plan.Dropped)
}

type memory map[string]string

func (m memory) Recall(english string) (string, bool) {
	text, ok := m[english]
	return text, ok
}

func TestALineAnotherCatalogueSaysIsNotSent(t *testing.T) {
	source := []translation.Line{line("20", "Ionian Islands"), line("21", "Atlantis")}

	plan := translation.PlanOf("fr", source, translation.EmptyCatalogue(), memory{"Ionian Islands": "Îles Ioniennes"})

	assert.Equal(t, []translation.Line{line("21", "Atlantis")}, plan.Send)
	assert.Equal(t, 1, plan.Recalled)
	outcome := translation.Apply(plan, map[translation.Key]string{"21": "Atlantide"})
	assert.Equal(t, map[translation.Key]string{"20": "Îles Ioniennes", "21": "Atlantide"}, outcome.Catalogue.Lines)
	assert.Equal(t, translation.FingerprintOf("Ionian Islands"), outcome.Catalogue.Fingerprints["20"])
}

func TestAnApplyKeepsWhatWasKeptAndFingerprintsWhatCameBack(t *testing.T) {
	source := []translation.Line{line("kept", "Send"), line("tiles", "{count, plural, one {# tile} other {# tiles}}")}
	current := translation.Catalogue{
		Lines:        map[translation.Key]string{"kept": "Envoyer"},
		Fingerprints: map[translation.Key]translation.Fingerprint{"kept": translation.FingerprintOf("Send")},
	}

	outcome := translation.Apply(translation.PlanOf("fr", source, current, translation.NoMemory{}), map[translation.Key]string{
		"tiles": "{count, plural, one {# case} many {# de cases} other {# cases}}", "kept": "Expédier",
	})

	assert.Equal(t, map[translation.Key]string{
		"kept": "Envoyer", "tiles": "{count, plural, one {# case} many {# de cases} other {# cases}}",
	}, outcome.Catalogue.Lines)
	assert.Equal(t, translation.FingerprintOf("{count, plural, one {# tile} other {# tiles}}"), outcome.Catalogue.Fingerprints["tiles"])
	assert.Equal(t, 1, outcome.Translated)
	assert.Empty(t, outcome.Refused)
}

func TestAnAnswerThatLosesASlotOrIsEmptyIsNotWritten(t *testing.T) {
	source := []translation.Line{line("took", "{player} took {count} tiles"), line("blank", "Chat"), line("silent", "Send")}

	outcome := translation.Apply(translation.PlanOf("fr", source, translation.EmptyCatalogue(), translation.NoMemory{}), map[translation.Key]string{
		"took": "{joueur} a pris {count} cases", "blank": " ",
	})

	assert.Empty(t, outcome.Catalogue.Lines)
	assert.Equal(t, []translation.Refusal{
		{Key: "took", Reason: "slots count:value joueur:value where the English has count:value player:value"},
		{Key: "blank", Reason: "empty"},
		{Key: "silent", Reason: "no answer"},
	}, outcome.Refused)
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
