// Package promoread is what the promo module's reads share, in the wire's words.
package promoread

import (
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/placeholders"
)

// Filled is a card as players read it: its words with the values the set plays.
func Filled(backgroundURL, headline, line string, played *planetv1.Rules) (*promov1.TeaserCard, error) {
	filledHeadline, err := placeholders.Fill(headline, played)
	if err != nil {
		return nil, fmt.Errorf("the headline does not fill: %w", err)
	}
	filledLine, err := placeholders.Fill(line, played)
	if err != nil {
		return nil, fmt.Errorf("the line does not fill: %w", err)
	}
	return &promov1.TeaserCard{BackgroundUrl: backgroundURL, Headline: filledHeadline, Line: filledLine}, nil
}
