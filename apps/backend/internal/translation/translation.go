// Package translation keeps keyed lines of English and their translations, each translation with the fingerprint of
// the English it was made from, so a translator is sent only the lines that are new or whose English changed.
package translation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

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

// What a language needs: the lines to send, and the lines it keeps as they are.
type Plan struct {
	Language Language
	Send     []Line
	// A line with no fingerprint was written by hand: it is taken as the translation of today's English.
	Adopted int
	Kept    int
	// Lines whose key English no longer has.
	Dropped int
	kept    Catalogue
}

func PlanOf(language Language, source []Line, current Catalogue) Plan {
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
			plan.Send = append(plan.Send, line)
			continue
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

// Spend is what the calls cost, in the model's tokens.
type Spend struct {
	InputTokens  int64
	OutputTokens int64
}

func (s Spend) Plus(other Spend) Spend {
	return Spend{InputTokens: s.InputTokens + other.InputTokens, OutputTokens: s.OutputTokens + other.OutputTokens}
}

// A Translator answers each line it is sent, keyed as it was sent.
type Translator interface {
	Translate(ctx context.Context, language Language, lines []Line) (map[Key]string, Spend, error)
}

// A translation that does not keep its English's slots is not written: the line is sent again next time.
type Refusal struct {
	Key    Key
	Reason string
}

type Outcome struct {
	Plan       Plan
	Catalogue  Catalogue
	Translated int
	Refused    []Refusal
	Spend      Spend
}

// Brings one language's catalogue up to its English, sending the translator only the plan's lines.
func Translate(ctx context.Context, plan Plan, translator Translator) (Outcome, error) {
	outcome := Outcome{Plan: plan, Catalogue: plan.kept}
	if len(plan.Send) == 0 {
		return outcome, nil
	}
	answered, spend, err := translator.Translate(ctx, plan.Language, plan.Send)
	outcome.Spend = spend
	if err != nil {
		return outcome, fmt.Errorf("failed to translate %d lines into %s: %w", len(plan.Send), plan.Language, err)
	}
	for _, line := range plan.Send {
		text, ok := answered[line.Key]
		if !ok {
			outcome.Refused = append(outcome.Refused, Refusal{Key: line.Key, Reason: "no answer"})
			continue
		}
		if reason := SlotsDiffer(line.English, text); reason != "" {
			outcome.Refused = append(outcome.Refused, Refusal{Key: line.Key, Reason: reason})
			continue
		}
		outcome.take(line, text)
		outcome.Translated++
	}
	return outcome, nil
}

func (o *Outcome) take(line Line, text string) {
	o.Catalogue.Lines[line.Key] = text
	o.Catalogue.Fingerprints[line.Key] = FingerprintOf(line.English)
}
