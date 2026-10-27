package zernio

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpzernio"
)

// TikTok's trending tracks in the country, rank 1 first, read for the one connected TikTok account: the library
// answers only for an account connected through the TikTok for Business app. Each with the id of its trending excerpt,
// and the whole track's only when it has none: with the whole track's, viewers have seen "This song is not available in
// your country" on the sound's page (Zernio's own note on the operation, observed from Germany, 2026-10-06).
func (s *Service) Trending(ctx context.Context, country string) ([]posts.Track, error) {
	accounts, refused, err := s.accounts(ctx, []posts.Platform{posts.TikTok})
	if err != nil {
		return nil, err
	}
	if why := refused[posts.TikTok]; why != nil {
		return nil, why
	}

	answer, err := s.api.ListTikTokCommercialMusicWithResponse(ctx, accounts[posts.TikTok],
		&cpzernio.ListTikTokCommercialMusicParams{CountryCode: &country})
	if err != nil {
		return nil, unreachable(err)
	}
	if answer.JSON200 == nil {
		return nil, refusal(answer.HTTPResponse, answer.Body, "the commercial music library")
	}
	if answer.JSON200.Tracks == nil {
		return nil, errors.New("zernio answered the commercial music library with no tracks in it")
	}

	tracks := make([]posts.Track, 0, len(*answer.JSON200.Tracks))
	for _, listed := range *answer.JSON200.Tracks {
		track := posts.Track{
			ID: valueOf(listed.Id), Title: valueOf(listed.Name), Artist: valueOf(listed.Artist), Rank: valueOf(listed.Rank),
			Seconds: valueOf(listed.DurationSec), Genres: valueOf(listed.Genres),
		}
		if listed.Clip != nil && valueOf(listed.Clip.Id) != "" {
			track.ID, track.Excerpt, track.Seconds = *listed.Clip.Id, true, valueOf(listed.Clip.DurationSec)
		}
		tracks = append(tracks, track)
	}
	return tracks, nil
}
