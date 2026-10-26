//go:build testing

package bans

import "context"

func (b *Banner) EnforcedSentence(ctx context.Context, scope string) (Sentence, bool, error) {
	if !b.rules().Enforce {
		return Sentence{}, false, nil
	}

	return b.Sentence(ctx, scope)
}

func (b *Book) EnforcedSentence(ctx context.Context, caller Caller) (Sentence, bool, error) {
	return b.sentenceOf(ctx, caller, (*Banner).EnforcedSentence)
}
