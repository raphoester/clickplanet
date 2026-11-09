// The numbers of the clips' posts, read from the posting service, Zernio, once a day: the one package of the job that
// still knows the service, until the snapshots move to the server's publishing module.
package zernio_source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/metadata"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/usecases/take_snapshot_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpzernio"
)

const Production = "https://zernio.com/api"

const (
	inFlight = "idempotency_conflict"
	// About fifteen posts a day: a page this long holds weeks of them.
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

var _ take_snapshot_usecase.Source = (*Service)(nil)

// Whether zernio gives analytics at all, and each TikTok account with the lane it is connected through.
func (s *Service) Accounts(ctx context.Context) (stats.Accounts, error) {
	answer, err := s.api.ListAccountsWithResponse(ctx, &cpzernio.ListAccountsParams{ExcludeHidden: new(true)})
	if err != nil {
		return stats.Accounts{}, unreachable(err)
	}
	if answer.JSON200 == nil {
		return stats.Accounts{}, refusal(answer.HTTPResponse, answer.Body, "the list of accounts")
	}

	accounts := stats.Accounts{AnalyticsAccess: answer.JSON200.HasAnalyticsAccess}
	for _, account := range answer.JSON200.Accounts {
		if !account.IsActive || account.Platform != cpzernio.SocialAccountPlatform(posts.TikTok) {
			continue
		}
		tiktok := stats.TikTokAccount{
			ID: account.UnderscoreId, Username: valueOf(account.Username), Type: string(valueOf(account.TiktokAccountType)),
		}
		tiktok.CommercialMusic, tiktok.CommercialMusicRefusal = s.commercialMusic(ctx, account.UnderscoreId)
		accounts.TikTok = append(accounts.TikTok, tiktok)
	}
	return accounts, nil
}

// TikTok's commercial music library answers only for an account connected through the TikTok for Business app.
func (s *Service) commercialMusic(ctx context.Context, account string) (bool, string) {
	answer, err := s.api.ListTikTokCommercialMusicWithResponse(ctx, account, &cpzernio.ListTikTokCommercialMusicParams{})
	if err != nil {
		return false, unreachable(err).Error()
	}
	if answer.StatusCode() == http.StatusOK {
		return true, ""
	}
	return false, refusal(answer.HTTPResponse, answer.Body, "the commercial music library").Error()
}

// The posts of the job are those with a story key: a post made by hand has none. fromDate is a date, so a few posts of
// the day before since may come too.
func (s *Service) Posts(ctx context.Context, since time.Time) ([]stats.Post, error) {
	var kept []stats.Post
	for page := 1; ; page++ {
		answer, err := s.api.ListPostsWithResponse(ctx, &cpzernio.ListPostsParams{
			FromDate: &openapi_types.Date{Time: since.UTC()}, Limit: new(pageOfPosts), Page: new(page),
		})
		if err != nil {
			return nil, unreachable(err)
		}
		if answer.JSON200 == nil || answer.JSON200.Posts == nil {
			return nil, refusal(answer.HTTPResponse, answer.Body, "the list of posts")
		}

		for _, post := range *answer.JSON200.Posts {
			if !keyed(post) || post.UnderscoreId == nil {
				continue
			}
			read, err := statsPostOf(post)
			if err != nil {
				return nil, err
			}
			kept = append(kept, read)
		}

		pages := 0
		if answer.JSON200.Pagination != nil {
			pages = valueOf(answer.JSON200.Pagination.Pages)
		}
		if page >= pages || len(*answer.JSON200.Posts) < pageOfPosts {
			return kept, nil
		}
	}
}

func statsPostOf(post cpzernio.Post) (stats.Post, error) {
	metadata, err := json.Marshal(post.Metadata)
	if err != nil {
		return stats.Post{}, fmt.Errorf("failed to write back the metadata of the post %s: %w", *post.UnderscoreId, err)
	}
	read := stats.Post{ID: stats.PostID(*post.UnderscoreId), Metadata: metadata}
	if post.Platforms == nil {
		return read, nil
	}
	for _, target := range *post.Platforms {
		read.Targets = append(read.Targets, stats.Target{
			Platform: valueOf(target.Platform), AccountID: accountOf(target),
			PlatformPostID: valueOf(target.PlatformPostId), URL: valueOf(target.PlatformPostUrl),
			PublishedAt: publishedAt(target),
		})
	}
	return read, nil
}

// A target is out once its platform published it: its time, unless it failed or was taken down since.
func publishedAt(target cpzernio.PlatformTarget) time.Time {
	if valueOf(target.Status) != "published" || target.PublishedAt == nil {
		return time.Time{}
	}
	return *target.PublishedAt
}

// The account is its id, or the whole account when zernio filled it in.
func accountOf(target cpzernio.PlatformTarget) string {
	if target.AccountId == nil {
		return ""
	}
	if account, err := target.AccountId.AsSocialAccount(); err == nil && account.UnderscoreId != "" {
		return account.UnderscoreId
	}
	id, _ := target.AccountId.AsPlatformTargetAccountId0()
	return id
}

// A 202 is a post whose numbers are still being fetched, and a 424 one that failed on every platform: both are
// answers, kept as they came. A 402 is the add-on turned off.
func (s *Service) Analytics(ctx context.Context, post stats.PostID) (stats.Answer, error) {
	answer, err := s.api.GetAnalyticsWithResponse(ctx, &cpzernio.GetAnalyticsParams{PostId: new(string(post))})
	if err != nil {
		return stats.Answer{}, unreachable(err)
	}

	switch answer.StatusCode() {
	case http.StatusOK, http.StatusAccepted, http.StatusFailedDependency:
		return answerOf(answer.StatusCode(), answer.Body)
	case http.StatusPaymentRequired:
		return stats.Answer{}, stats.ErrNoAnalyticsAccess
	default:
		return stats.Answer{}, refusal(answer.HTTPResponse, answer.Body, "the analytics of the post "+string(post))
	}
}

func answerOf(status int, body []byte) (stats.Answer, error) {
	var said struct {
		Status            string            `json:"status"`
		SyncStatus        string            `json:"syncStatus"`
		Message           *string           `json:"message"`
		PlatformAnalytics []json.RawMessage `json:"platformAnalytics"`
	}
	if err := json.Unmarshal(body, &said); err != nil {
		return stats.Answer{}, fmt.Errorf("zernio answered %d to the analytics of a post with no analytics in it: %w", status, err)
	}

	answer := stats.Answer{HTTPStatus: status, Status: said.Status, SyncStatus: said.SyncStatus, Message: valueOf(said.Message)}
	for _, raw := range said.PlatformAnalytics {
		var entry struct {
			Platform  string `json:"platform"`
			AccountID string `json:"accountId"`
		}
		if err := json.Unmarshal(raw, &entry); err != nil {
			return stats.Answer{}, fmt.Errorf("zernio answered a platform's analytics that is not an object: %w", err)
		}
		answer.Entries = append(answer.Entries, stats.Entry{Platform: entry.Platform, AccountID: entry.AccountID, Raw: raw})
	}
	return answer, nil
}

// Up to the day asked, which YouTube may still revise for 3 days: the newest curve it has. Any answer but a busy one is
// kept as it came, a 412 that asks for YouTube to be connected again too.
func (s *Service) Retention(ctx context.Context, account, video string, asOf time.Time) (stats.Retention, error) {
	answer, err := s.api.GetYouTubeVideoRetentionWithResponse(ctx, &cpzernio.GetYouTubeVideoRetentionParams{
		VideoId: video, AccountId: account, ToDate: &openapi_types.Date{Time: asOf.UTC()},
	})
	if err != nil {
		return stats.Retention{}, unreachable(err)
	}

	status := answer.StatusCode()
	if status == http.StatusTooManyRequests || status/100 == 5 {
		return stats.Retention{}, refusal(answer.HTTPResponse, answer.Body, "the retention of the video "+video)
	}
	body := json.RawMessage(answer.Body)
	if !json.Valid(body) {
		body, _ = json.Marshal(string(answer.Body))
	}
	return stats.Retention{HTTPStatus: status, Body: body}, nil
}

// A post of the job carries its story's key: one made by hand has none.
func keyed(post cpzernio.Post) bool {
	if post.Metadata == nil {
		return false
	}
	_, keyed := metadata.KeyOf(*post.Metadata)
	return keyed
}

func valueOf[T any](pointer *T) T {
	if pointer == nil {
		var zero T
		return zero
	}
	return *pointer
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

// Without its URL: an address can carry a credential, and the logs of a public repo are public.
func unreachable(err error) error {
	var failed *url.Error
	if errors.As(err, &failed) {
		err = failed.Err
	}
	return &posts.BusyError{Reason: fmt.Sprintf("zernio could not be reached: %s", err)}
}
