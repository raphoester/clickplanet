// The one package of the server that knows the posting service, Zernio.
package zernio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpzernio"
)

const Production = "https://zernio.com/api"

const (
	everyone = "PUBLIC_TO_EVERYONE"
	gaming   = "20"
	inFlight = "idempotency_conflict"
	// About fifteen posts a day: a page this long holds the week a job reads, and more.
	pageOfPosts = 500
)

type Config struct {
	APIKey  string
	BaseURL string
}

func New(config Config, httpClient *http.Client) (*Service, error) {
	if config.APIKey == "" {
		return nil, errors.New("the zernio api key is empty")
	}
	if config.BaseURL == "" {
		config.BaseURL = Production
	}

	api, err := cpzernio.NewClientWithResponses(config.BaseURL, cpzernio.WithHTTPClient(httpClient),
		cpzernio.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+config.APIKey)
			return nil
		}))
	if err != nil {
		return nil, fmt.Errorf("failed to build the zernio client: %w", err)
	}
	return &Service{api: api}, nil
}

type Service struct {
	api *cpzernio.ClientWithResponses
}

// A link to the service's storage, which takes the file with no key, and the address the service then serves it at.
func (s *Service) MediaUpload(ctx context.Context, media posts.Media) (posts.MediaUpload, error) {
	answer, err := s.api.GetMediaPresignedUrlWithResponse(ctx, cpzernio.GetMediaPresignedUrlJSONRequestBody{
		Filename: media.FileName, ContentType: cpzernio.MediaContentType(media.ContentType), Size: new(int(media.Size)),
	})
	if err != nil {
		return posts.MediaUpload{}, unreachable(err)
	}
	if answer.JSON200 == nil {
		return posts.MediaUpload{}, refusal(answer.HTTPResponse, answer.Body, "the address to upload the video to")
	}
	if answer.JSON200.UploadUrl == nil || answer.JSON200.PublicUrl == nil {
		return posts.MediaUpload{}, errors.New("zernio gave no address to upload the video to")
	}
	return posts.MediaUpload{UploadURL: *answer.JSON200.UploadUrl, Media: posts.MediaURL(*answer.JSON200.PublicUrl)}, nil
}

// One post, with a target on each platform of the post that has one connected account; the others are refused, and
// with none left nothing is posted. The key is the post's: the same key is the same post there.
func (s *Service) Schedule(ctx context.Context, post posts.Post) (posts.Scheduled, error) {
	snapshot := post.TakeSnapshot()

	accounts, refused, err := s.accounts(ctx, snapshot.Platforms)
	if err != nil {
		return posts.Scheduled{}, err
	}
	targets, err := targetsOf(snapshot, accounts)
	if err != nil {
		return posts.Scheduled{}, err
	}
	if len(targets) == 0 {
		return posts.Scheduled{Refused: refused}, nil
	}

	var metadata *map[string]any
	if snapshot.Metadata != nil {
		metadata = &snapshot.Metadata
	}
	answer, err := s.api.CreatePostWithResponse(ctx,
		&cpzernio.CreatePostParams{IdempotencyKey: new(string(snapshot.Key))},
		cpzernio.CreatePostJSONRequestBody{
			Title:        &snapshot.Title,
			MediaItems:   &[]cpzernio.MediaItem{{Type: new(cpzernio.MediaItemTypeVideo), Url: new(string(snapshot.Media))}},
			Platforms:    &targets,
			ScheduledFor: new(snapshot.At.UTC()),
			Metadata:     metadata,
			Tags:         tagsOf(snapshot, accounts),
		})
	if err != nil {
		return posts.Scheduled{}, unreachable(err)
	}

	kept, err := keptPost(answer)
	if err != nil {
		return posts.Scheduled{}, err
	}
	if kept.UnderscoreId == nil || kept.Status == nil || *kept.Status != cpzernio.PostStatusScheduled {
		return posts.Scheduled{}, fmt.Errorf("zernio kept the post as %q, not %q", status(kept), cpzernio.PostStatusScheduled)
	}
	for platform := range accounts {
		if !on(kept, platform) {
			refused[platform] = errors.New("zernio kept the post without it")
		}
	}
	return posts.Scheduled{ID: posts.PostID(*kept.UnderscoreId), Refused: refused}, nil
}

func targetsOf(snapshot posts.PostSnapshot, accounts map[calendar.Platform]string) ([]cpzernio.CreatePostJSONBody_Platforms, error) {
	targets := make([]cpzernio.CreatePostJSONBody_Platforms, 0, len(accounts))
	for _, platform := range snapshot.Platforms {
		account, connected := accounts[platform]
		if !connected {
			continue
		}
		target, err := targetOf(snapshot, platform, account)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func targetOf(snapshot posts.PostSnapshot, platform calendar.Platform, account string) (cpzernio.CreatePostJSONBody_Platforms, error) {
	settings := &cpzernio.CreatePostJSONBody_Platforms_PlatformSpecificData{}
	target := cpzernio.CreatePostJSONBody_Platforms{Platform: string(platform), AccountId: account, PlatformSpecificData: settings}

	var err error
	switch platform {
	case calendar.TikTok:
		target.CustomContent = &snapshot.TikTok.Caption
		err = settings.FromTikTokPlatformData(tiktokSettings())
	case calendar.YouTube:
		target.CustomContent = &snapshot.YouTube.Description
		// Not made for kids, sent every time: YouTube may block the views of a video with no answer, and made for kids
		// turns its comments off for good (the owner's legal choice, 2026-10-17).
		err = settings.FromYouTubePlatformData(cpzernio.YouTubePlatformData{
			Title: &snapshot.YouTube.Title, Visibility: new(cpzernio.YouTubePlatformDataVisibilityPublic),
			MadeForKids: new(false), CategoryId: new(gaming),
		})
	default:
		return target, fmt.Errorf("the game does not post to %q", platform)
	}
	if err != nil {
		return target, fmt.Errorf("failed to write the settings of %s: %w", platform, err)
	}
	return target, nil
}

// Public, with comments, duets and stitches.
func tiktokSettings() cpzernio.TikTokPlatformData {
	return cpzernio.TikTokPlatformData{
		PrivacyLevel: new(everyone), AllowComment: new(true), AllowDuet: new(true), AllowStitch: new(true),
		ContentPreviewConfirmed: new(true), ExpressConsentGiven: new(true),
	}
}

// Zernio sends them to YouTube alone.
func tagsOf(snapshot posts.PostSnapshot, accounts map[calendar.Platform]string) *[]string {
	if _, connected := accounts[calendar.YouTube]; !connected {
		return nil
	}
	return &snapshot.YouTube.Tags
}

func on(post cpzernio.Post, platform calendar.Platform) bool {
	return post.Platforms != nil && slices.ContainsFunc(*post.Platforms, func(target cpzernio.PlatformTarget) bool {
		return valueOf(target.Platform) == string(platform)
	})
}

// Every post, by hand or not, on each platform it is to go out on, or went out on, at that platform's time: Zernio
// moves a TikTok target held over the cap to when it will go out. A draft, a post cancelled or failed, and a target
// that failed or was unpublished count for nothing.
func (s *Service) Posted(ctx context.Context, since time.Time) (calendar.Feed, error) {
	listed, err := s.list(ctx, since)
	if err != nil {
		return nil, err
	}

	var feed calendar.Feed
	for _, post := range listed {
		switch valueOf(post.Status) {
		case cpzernio.PostStatusDraft, cpzernio.PostStatusCancelled, cpzernio.PostStatusFailed:
			continue
		}
		feed = append(feed, postingsOf(post)...)
	}
	return feed, nil
}

// One for each time the post's targets go out at, most often one.
func postingsOf(post cpzernio.Post) []calendar.Posting {
	if post.Platforms == nil {
		return nil
	}
	var postings []calendar.Posting
	for _, target := range *post.Platforms {
		platform, err := calendar.PlatformsOf(valueOf(target.Platform))
		if err != nil {
			continue
		}
		switch valueOf(target.Status) {
		case "failed", "cancelled":
			continue
		}
		at := firstTimeOf(target.ScheduledFor, post.ScheduledFor, target.PublishedAt)
		if at.IsZero() {
			continue
		}
		i := slices.IndexFunc(postings, func(posting calendar.Posting) bool { return posting.At.Equal(at) })
		if i < 0 {
			postings = append(postings, calendar.Posting{ID: valueOf(post.UnderscoreId), At: at})
			i = len(postings) - 1
		}
		postings[i].Platforms = append(postings[i].Platforms, platform...)
	}
	return postings
}

// A post published at once by hand may have no scheduled time, only the time it went out.
func firstTimeOf(times ...*time.Time) time.Time {
	for _, at := range times {
		if at != nil && !at.IsZero() {
			return *at
		}
	}
	return time.Time{}
}

// Every post as the service holds it, by hand or not, with the metadata it was posted with, as ListPosts answers it.
func (s *Service) Posts(ctx context.Context, since time.Time) ([]*publishingv1.Post, error) {
	listed, err := s.list(ctx, since)
	if err != nil {
		return nil, err
	}

	answered := make([]*publishingv1.Post, 0, len(listed))
	for _, post := range listed {
		read := &publishingv1.Post{
			PostId: valueOf(post.UnderscoreId), Title: valueOf(post.Title), Status: string(valueOf(post.Status)),
			Failure: failureOf(post),
		}
		if post.ScheduledFor != nil {
			read.ScheduledFor = timestamppb.New(*post.ScheduledFor)
		}
		if post.Metadata != nil {
			if read.Metadata, err = structpb.NewStruct(*post.Metadata); err != nil {
				return nil, fmt.Errorf("the metadata of the post %s cannot be read: %w", read.GetPostId(), err)
			}
		}
		answered = append(answered, read)
	}
	return answered, nil
}

// fromDate is a date: a caller filters to the hour.
func (s *Service) list(ctx context.Context, since time.Time) ([]cpzernio.Post, error) {
	answer, err := s.api.ListPostsWithResponse(ctx, &cpzernio.ListPostsParams{
		FromDate: &openapi_types.Date{Time: since.UTC()}, Limit: new(pageOfPosts),
	})
	if err != nil {
		return nil, unreachable(err)
	}
	if answer.JSON200 == nil || answer.JSON200.Posts == nil {
		return nil, refusal(answer.HTTPResponse, answer.Body, "the list of posts")
	}
	return *answer.JSON200.Posts, nil
}

func failureOf(post cpzernio.Post) string {
	if post.Platforms == nil {
		return ""
	}
	var failures []string
	for _, target := range *post.Platforms {
		if message := valueOf(target.ErrorMessage); message != "" {
			failures = append(failures, valueOf(target.Platform)+": "+message)
		}
	}
	return strings.Join(failures, "; ")
}

func valueOf[T any](pointer *T) T {
	if pointer == nil {
		var zero T
		return zero
	}
	return *pointer
}

// A 201 is a new post, and a 200 the post an earlier try with the same key already made.
func keptPost(answer *cpzernio.CreatePostResponse) (cpzernio.Post, error) {
	switch {
	case answer.JSON201 != nil && answer.JSON201.Post != nil:
		return *answer.JSON201.Post, nil
	case answer.JSON200 != nil:
		replayed, err := answer.JSON200.AsPostCreateResponse()
		if err != nil || replayed.Post == nil {
			return cpzernio.Post{}, errors.New("zernio answered 200 to the post with no post in it")
		}
		return *replayed.Post, nil
	default:
		return cpzernio.Post{}, refusal(answer.HTTPResponse, answer.Body, "the post")
	}
}

func status(post cpzernio.Post) cpzernio.PostStatus {
	if post.Status == nil {
		return ""
	}
	return *post.Status
}

// The one connected account of each platform. A platform with none or two is refused, and the post still goes to the
// others: no account id is configured, so the adapter never guesses between two.
func (s *Service) accounts(ctx context.Context, platforms []calendar.Platform) (map[calendar.Platform]string, map[calendar.Platform]error, error) {
	answer, err := s.api.ListAccountsWithResponse(ctx, &cpzernio.ListAccountsParams{ExcludeHidden: new(true)})
	if err != nil {
		return nil, nil, unreachable(err)
	}
	if answer.JSON200 == nil {
		return nil, nil, refusal(answer.HTTPResponse, answer.Body, "the list of accounts")
	}

	connected := map[calendar.Platform][]string{}
	for _, account := range answer.JSON200.Accounts {
		if account.IsActive {
			connected[calendar.Platform(account.Platform)] = append(connected[calendar.Platform(account.Platform)], account.UnderscoreId)
		}
	}

	accounts, refused := map[calendar.Platform]string{}, map[calendar.Platform]error{}
	for _, platform := range platforms {
		if ids := connected[platform]; len(ids) != 1 {
			refused[platform] = fmt.Errorf("zernio has %d connected %s accounts, and the game posts to exactly one", len(ids), platform)
			continue
		}
		accounts[platform] = connected[platform][0]
	}
	return accounts, refused, nil
}

// Busy for what zernio may take later: a rate limit, a server error, a try made while the one before it is in flight.
func refusal(res *http.Response, body []byte, what string) error {
	var said struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	_ = json.Unmarshal(body, &said)

	reason := fmt.Sprintf("zernio answered %d to %s: %s", res.StatusCode, what, said.Error)
	if said.Code != inFlight && res.StatusCode != http.StatusTooManyRequests && res.StatusCode/100 != 5 {
		return errors.New(reason)
	}

	seconds, _ := strconv.Atoi(res.Header.Get("Retry-After"))
	return &posts.BusyError{RetryAfter: time.Duration(seconds) * time.Second, Reason: reason}
}

// Without its URL: an address can carry a credential, and the logs say what an error says.
func unreachable(err error) error {
	var failed *url.Error
	if errors.As(err, &failed) {
		err = failed.Err
	}
	return &posts.BusyError{Reason: fmt.Sprintf("zernio could not be reached: %s", err)}
}
