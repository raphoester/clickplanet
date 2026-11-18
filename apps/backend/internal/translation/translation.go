// Package translation keeps keyed lines of English and their translations, each translation with the fingerprint of
// the English it was made from, so a translator is given only the lines that are new or whose English changed.
package translation

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Key string

// A BCP 47 tag, as the catalogue's files are named: "fr", "pt-BR".
type Language string

// An English line to translate, with what a translator needs to know of it: where it shows, what fills its slots.
type Line struct {
	Key     Key
	English string
	Context string
}

type Fingerprint string

// The English a translation was made from, in 16 hex digits: a line whose English did not change is never sent again,
// so a fix made by hand stays.
func FingerprintOf(english string) Fingerprint {
	sum := sha256.Sum256([]byte(english))
	return Fingerprint(hex.EncodeToString(sum[:8]))
}

// One language's lines, each with the fingerprint of the English it came from.
type Catalogue struct {
	Lines        map[Key]string
	Fingerprints map[Key]Fingerprint
}

func EmptyCatalogue() Catalogue {
	return Catalogue{Lines: map[Key]string{}, Fingerprints: map[Key]Fingerprint{}}
}

// What a language already says for an English outside the catalogue planned: a line it knows is not sent.
type Memory interface {
	Recall(english string) (string, bool)
}

// A catalogue whose lines are said nowhere else.
type NoMemory struct{}

func (NoMemory) Recall(string) (string, bool) {
	return "", false
}

// What a language needs: the lines to send, and the lines it keeps as they are.
type Plan struct {
	Language Language
	Send     []Line
	// A line with no fingerprint was written by hand: it is taken as the translation of today's English.
	Adopted  int
	Kept     int
	Recalled int
	// Lines whose key English no longer has.
	Dropped int
	kept    Catalogue
}

func PlanOf(language Language, source []Line, current Catalogue, memory Memory) Plan {
	plan := Plan{Language: language, kept: EmptyCatalogue()}
	inSource := cpcolls.NewSetWithCapacity[Key](len(source))
	for _, line := range source {
		inSource.Add(line.Key)
		fingerprint := FingerprintOf(line.English)
		text, held := current.Lines[line.Key]
		was, fingerprinted := current.Fingerprints[line.Key]
		switch {
		case held && fingerprinted && was == fingerprint:
			plan.Kept++
		case held && !fingerprinted:
			plan.Adopted++
		default:
			recalled, known := memory.Recall(line.English)
			if !known {
				plan.Send = append(plan.Send, line)
				continue
			}
			text = recalled
			plan.Recalled++
		}
		plan.kept.Lines[line.Key] = text
		plan.kept.Fingerprints[line.Key] = fingerprint
	}
	for key := range current.Lines {
		if !inSource.Contains(key) {
			plan.Dropped++
		}
	}
	return plan
}

// A translation that is empty, or does not keep its English's slots, is not written: the line is planned again.
type Refusal struct {
	Key    Key
	Reason string
}

type Outcome struct {
	Plan       Plan
	Catalogue  Catalogue
	Translated int
	Refused    []Refusal
}

// Brings one language's catalogue up to its English with the answers to the plan's lines.
func Apply(plan Plan, answers map[Key]string) Outcome {
	outcome := Outcome{Plan: plan, Catalogue: plan.kept}
	for _, line := range plan.Send {
		text, ok := answers[line.Key]
		if !ok {
			outcome.Refused = append(outcome.Refused, Refusal{Key: line.Key, Reason: "no answer"})
			continue
		}
		if strings.TrimSpace(text) == "" {
			outcome.Refused = append(outcome.Refused, Refusal{Key: line.Key, Reason: "empty"})
			continue
		}
		if reason := SlotsDiffer(line.English, text); reason != "" {
			outcome.Refused = append(outcome.Refused, Refusal{Key: line.Key, Reason: reason})
			continue
		}
		outcome.Catalogue.Lines[line.Key] = text
		outcome.Catalogue.Fingerprints[line.Key] = FingerprintOf(line.English)
		outcome.Translated++
	}
	return outcome
}
