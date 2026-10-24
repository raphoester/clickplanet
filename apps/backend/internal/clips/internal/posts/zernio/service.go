package zernio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpzernio"
)

const Production = "https://zernio.com/api"

const (
	maxBodyBytes = 1 << 16
	everyone     = "PUBLIC_TO_EVERYONE"
	gaming       = "20"
	inFlight     = "idempotency_conflict"

	flagKey      = "storyFlag"
	continentKey = "storyContinent"
	countriesKey = "storyCountries"
	callKey      = "callToAction"
	// Absent on a post of what a flag did, and on every post made before them.
	angleKey = "storyAngle"
	roundKey = "storyRound"
	// What the post was, to set beside how it did: a clips.v1.PostRecord.
	recordKey = "record"
	// About ten posts a day: a page this long holds the week the choice reads, and more.
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
	return &Service{api: api, http: httpClient}, nil
}

type Service struct {
	api *cpzernio.ClientWithResponses
	// For the upload alone: the storage is not zernio's API, and the key never goes there.
	http *http.Client
}

var _ posts.Service = (*Service)(nil)

func (s *Service) Schedule(ctx context.Context, post posts.Post) (posts.Scheduled, error) {
	snapshot := post.TakeSnapshot()

	video, err := os.ReadFile(string(snapshot.Video)) //nolint:gosec // the file of a clip this job rendered.
	if err != nil {
		return posts.Scheduled{}, fmt.Errorf("failed to read the video: %w", err)
	}

	accounts, refused, err := s.accounts(ctx, snapshot.Platforms)
	if err != nil {
		return posts.Scheduled{}, err
	}
	targets, err := targetsOf(snapshot, accounts)
	if err != nil {
		return posts.Scheduled{}, err
	}
	metadata, err := metadataOf(snapshot)
	if err != nil {
		return posts.Scheduled{}, err
	}
	if len(targets) == 0 {
		return posts.Scheduled{Refused: refused}, nil
	}
	media, err := s.upload(ctx, filepath.Base(string(snapshot.Video)), video)
	if err != nil {
		return posts.Scheduled{}, err
	}

	// The same video is the same post: a run made again does not post it twice.
	key := sha256.Sum256(video)

	answer, err := s.api.CreatePostWithResponse(ctx,
		&cpzernio.CreatePostParams{IdempotencyKey: new(hex.EncodeToString(key[:]))},
		cpzernio.CreatePostJSONRequestBody{
			Title:        &snapshot.Title,
			MediaItems:   &[]cpzernio.MediaItem{{Type: new(cpzernio.MediaItemTypeVideo), Url: &media}},
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

func targetsOf(snapshot posts.PostSnapshot, accounts map[posts.Platform]string) ([]cpzernio.CreatePostJSONBody_Platforms, error) {
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

func targetOf(snapshot posts.PostSnapshot, platform posts.Platform, account string) (cpzernio.CreatePostJSONBody_Platforms, error) {
	settings := &cpzernio.CreatePostJSONBody_Platforms_PlatformSpecificData{}
	target := cpzernio.CreatePostJSONBody_Platforms{Platform: string(platform), AccountId: account, PlatformSpecificData: settings}

	var err error
	switch platform {
	case posts.TikTok:
		target.CustomContent = &snapshot.TikTok.Caption
		err = settings.FromTikTokPlatformData(cpzernio.TikTokPlatformData{
			PrivacyLevel: new(everyone), AllowComment: new(true), AllowDuet: new(true), AllowStitch: new(true),
			ContentPreviewConfirmed: new(true), ExpressConsentGiven: new(true),
		})
	case posts.YouTube:
		target.CustomContent = &snapshot.YouTube.Description
		err = settings.FromYouTubePlatformData(cpzernio.YouTubePlatformData{
			Title: &snapshot.YouTube.Title, Visibility: new(cpzernio.YouTubePlatformDataVisibilityPublic),
			MadeForKids: new(false), CategoryId: new(gaming),
		})
	default:
		return target, fmt.Errorf("the job does not post to %q", platform)
	}
	if err != nil {
		return target, fmt.Errorf("failed to write the settings of %s: %w", platform, err)
	}
	return target, nil
}

// What the choice reads back of the post: its key and its countries; and the call to act, to compare the calls on the
// posts' numbers. Beside them, what the post was, as one document (recordKey). Never published.
func metadataOf(snapshot posts.PostSnapshot) (*map[string]any, error) {
	record, err := recordOf(snapshot)
	if err != nil {
		return nil, err
	}
	metadata := map[string]any{
		flagKey: snapshot.Key.Flag, continentKey: snapshot.Key.Continent, countriesKey: snapshot.Countries,
		callKey: snapshot.CallToAction, recordKey: record,
	}
	if snapshot.Key.Angle != posts.Deeds {
		metadata[angleKey] = string(snapshot.Key.Angle)
	}
	if snapshot.Key.Round != "" {
		metadata[roundKey] = snapshot.Key.Round
	}
	return &metadata, nil
}

// A clips.v1.PostRecord as protojson writes it, under one key: what the clip was as the generator wrote it, and what
// this job made of it. Its shape is the proto's, not Zernio's, and it grows there.
func recordOf(snapshot posts.PostSnapshot) (map[string]any, error) {
	record := &clipsv1.PostRecord{
		Schema: string((&clipsv1.PostRecord{}).ProtoReflect().Descriptor().FullName()),
		Post: &clipsv1.PostFacts{
			Teaser: snapshot.Teaser, CallToAction: snapshot.CallToAction, ScheduledAt: timestamppb.New(snapshot.At),
			RunPosition: uint32(snapshot.RunPosition), RunSize: uint32(snapshot.RunSize), //nolint:gosec // 2 a run.
		},
	}
	if len(snapshot.Record) != 0 {
		record.Clip = &clipsv1.ClipRecord{}
		if err := protojson.Unmarshal(snapshot.Record, record.Clip); err != nil {
			return nil, fmt.Errorf("what the clip was is not a clips.v1.ClipRecord: %w", err)
		}
	}
	written, err := protojson.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to write what the post was: %w", err)
	}
	var document map[string]any
	if err := json.Unmarshal(written, &document); err != nil {
		return nil, fmt.Errorf("failed to read back what the post was: %w", err)
	}
	return document, nil
}

// Zernio sends them to YouTube alone.
func tagsOf(snapshot posts.PostSnapshot, accounts map[posts.Platform]string) *[]string {
	if _, connected := accounts[posts.YouTube]; !connected {
		return nil
	}
	return &snapshot.YouTube.Tags
}

func on(post cpzernio.Post, platform posts.Platform) bool {
	return post.Platforms != nil && slices.ContainsFunc(*post.Platforms, func(target cpzernio.PlatformTarget) bool {
		return valueOf(target.Platform) == string(platform)
	})
}

// The posts made by hand carry no story key, and are left out.
func (s *Service) Told(ctx context.Context, since time.Time) ([]posts.Telling, error) {
	listed, err := s.list(ctx, since)
	if err != nil {
		return nil, err
	}

	told := make([]posts.Telling, 0, len(listed))
	for _, post := range listed {
		key, keyed := keyOf(post)
		if !keyed || post.ScheduledFor == nil {
			continue
		}
		told = append(told, posts.Telling{
			Key: key, Countries: countriesOf(post), At: *post.ScheduledFor, Title: valueOf(post.Title),
			State: string(valueOf(post.Status)), Failure: failureOf(post),
		})
	}
	return told, nil
}

// Every post, by hand or not, on each platform it is to go out on, or went out on, at that platform's time: Zernio
// moves a TikTok target held over the cap to when it will go out. A draft, a post cancelled or failed, and a target
// that failed or was unpublished count for nothing.
func (s *Service) Posted(ctx context.Context, since time.Time) (posts.Feed, error) {
	listed, err := s.list(ctx, since)
	if err != nil {
		return nil, err
	}

	var feed posts.Feed
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
func postingsOf(post cpzernio.Post) []posts.Posting {
	if post.Platforms == nil {
		return nil
	}
	var postings []posts.Posting
	for _, target := range *post.Platforms {
		platform, err := posts.PlatformsOf(valueOf(target.Platform))
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
		i := slices.IndexFunc(postings, func(posting posts.Posting) bool { return posting.At.Equal(at) })
		if i < 0 {
			postings = append(postings, posts.Posting{At: at})
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

func keyOf(post cpzernio.Post) (posts.StoryKey, bool) {
	if post.Metadata == nil {
		return posts.StoryKey{}, false
	}
	flag, _ := (*post.Metadata)[flagKey].(string)
	continent, _ := (*post.Metadata)[continentKey].(string)
	angle, _ := (*post.Metadata)[angleKey].(string)
	round, _ := (*post.Metadata)[roundKey].(string)
	return posts.StoryKey{Flag: flag, Continent: continent, Angle: posts.Angle(angle), Round: round}, flag != "" && continent != ""
}

func countriesOf(post cpzernio.Post) []string {
	if post.Metadata == nil {
		return nil
	}
	kept, _ := (*post.Metadata)[countriesKey].([]any)
	var countries []string
	for _, value := range kept {
		if country, ok := value.(string); ok && country != "" {
			countries = append(countries, country)
		}
	}
	return countries
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

// The one connected account of each platform. A platform with none or two is refused, and the post still goes to the others.
func (s *Service) accounts(ctx context.Context, platforms []posts.Platform) (map[posts.Platform]string, map[posts.Platform]error, error) {
	answer, err := s.api.ListAccountsWithResponse(ctx, &cpzernio.ListAccountsParams{ExcludeHidden: new(true)})
	if err != nil {
		return nil, nil, unreachable(err)
	}
	if answer.JSON200 == nil {
		return nil, nil, refusal(answer.HTTPResponse, answer.Body, "the list of accounts")
	}

	connected := map[posts.Platform][]string{}
	for _, account := range answer.JSON200.Accounts {
		if account.IsActive {
			connected[posts.Platform(account.Platform)] = append(connected[posts.Platform(account.Platform)], account.UnderscoreId)
		}
	}

	accounts, refused := map[posts.Platform]string{}, map[posts.Platform]error{}
	for _, platform := range platforms {
		if ids := connected[platform]; len(ids) != 1 {
			refused[platform] = fmt.Errorf("zernio has %d connected %s accounts, and the job posts to exactly one", len(ids), platform)
			continue
		}
		accounts[platform] = connected[platform][0]
	}
	return accounts, refused, nil
}

func (s *Service) upload(ctx context.Context, name string, video []byte) (string, error) {
	answer, err := s.api.GetMediaPresignedUrlWithResponse(ctx, cpzernio.GetMediaPresignedUrlJSONRequestBody{
		Filename: name, ContentType: cpzernio.MediaContentTypeVideomp4, Size: new(len(video)),
	})
	if err != nil {
		return "", unreachable(err)
	}
	if answer.JSON200 == nil {
		return "", refusal(answer.HTTPResponse, answer.Body, "the address to upload the video to")
	}
	if answer.JSON200.UploadUrl == nil || answer.JSON200.PublicUrl == nil {
		return "", errors.New("zernio gave no address to upload the video to")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, *answer.JSON200.UploadUrl, bytes.NewReader(video))
	if err != nil {
		return "", errors.New("zernio gave an address to upload the video to that is not a URL")
	}
	req.Header.Set("Content-Type", string(cpzernio.MediaContentTypeVideomp4))

	res, err := s.http.Do(req)
	if err != nil {
		return "", unreachable(err)
	}
	defer func() { _ = res.Body.Close() }()
	said, _ := io.ReadAll(io.LimitReader(res.Body, maxBodyBytes))

	if res.StatusCode/100 != 2 {
		return "", refusal(res, said, "the upload of the video")
	}
	return *answer.JSON200.PublicUrl, nil
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

// Without its URL: the address a video is uploaded to is a credential, and the logs are public.
func unreachable(err error) error {
	var failed *url.Error
	if errors.As(err, &failed) {
		err = failed.Err
	}
	return &posts.BusyError{Reason: fmt.Sprintf("zernio could not be reached: %s", err)}
}
