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
	"strconv"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
)

const Production = "https://zernio.com/api/v1"

const (
	maxBodyBytes = 1 << 16
	platform     = "tiktok"
	videoType    = "video/mp4"
	scheduled    = "scheduled"
)

type Config struct {
	APIKey string
}

func New(config Config, baseURL string, httpClient *http.Client) (*Service, error) {
	if config.APIKey == "" {
		return nil, errors.New("the zernio api key is empty")
	}
	return &Service{base: baseURL, key: config.APIKey, http: httpClient}, nil
}

type Service struct {
	base string
	key  string
	http *http.Client
}

var _ posts.Service = (*Service)(nil)

func (s *Service) Schedule(ctx context.Context, post posts.Post) (posts.PostID, error) {
	snapshot := post.TakeSnapshot()

	video, err := os.ReadFile(string(snapshot.Video)) //nolint:gosec // the file of a clip this job rendered.
	if err != nil {
		return "", fmt.Errorf("failed to read the video: %w", err)
	}

	account, err := s.account(ctx)
	if err != nil {
		return "", err
	}
	media, err := s.upload(ctx, filepath.Base(string(snapshot.Video)), video)
	if err != nil {
		return "", err
	}

	// The same video is the same post: a run made again does not post it twice.
	key := sha256.Sum256(video)

	var created created
	err = s.call(ctx, http.MethodPost, "/posts", map[string]string{"Idempotency-Key": hex.EncodeToString(key[:])}, createPost{
		Title:        snapshot.Title,
		Content:      snapshot.Caption,
		MediaItems:   []mediaItem{{Type: "video", URL: media}},
		Platforms:    []target{{Platform: platform, AccountID: account}},
		ScheduledFor: snapshot.At.UTC().Format(time.RFC3339),
		TikTok: tiktokSettings{
			PrivacyLevel: "PUBLIC_TO_EVERYONE", AllowComment: true, AllowDuet: true, AllowStitch: true,
			ContentPreviewConfirmed: true, ExpressConsentGiven: true,
		},
	}, &created)
	if err != nil {
		return "", err
	}
	if created.Post.ID == "" || created.Post.Status != scheduled {
		return "", fmt.Errorf("zernio kept the post as %q, not %q", created.Post.Status, scheduled)
	}
	return posts.PostID(created.Post.ID), nil
}

type createPost struct {
	Title        string         `json:"title"`
	Content      string         `json:"content"`
	MediaItems   []mediaItem    `json:"mediaItems"`
	Platforms    []target       `json:"platforms"`
	ScheduledFor string         `json:"scheduledFor"`
	TikTok       tiktokSettings `json:"tiktokSettings"`
}

type mediaItem struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type target struct {
	Platform  string `json:"platform"`
	AccountID string `json:"accountId"`
}

type tiktokSettings struct {
	PrivacyLevel            string `json:"privacyLevel"`
	AllowComment            bool   `json:"allowComment"`
	AllowDuet               bool   `json:"allowDuet"`
	AllowStitch             bool   `json:"allowStitch"`
	ContentPreviewConfirmed bool   `json:"contentPreviewConfirmed"`
	ExpressConsentGiven     bool   `json:"expressConsentGiven"`
}

type created struct {
	Post struct {
		ID     string `json:"_id"`
		Status string `json:"status"`
	} `json:"post"`
}

func (s *Service) account(ctx context.Context) (string, error) {
	var listed struct {
		Accounts []struct {
			ID       string `json:"_id"`
			Platform string `json:"platform"`
			IsActive bool   `json:"isActive"`
		} `json:"accounts"`
	}
	if err := s.call(ctx, http.MethodGet, "/accounts?platform="+platform, nil, nil, &listed); err != nil {
		return "", err
	}

	var connected []string
	for _, account := range listed.Accounts {
		if account.Platform == platform && account.IsActive {
			connected = append(connected, account.ID)
		}
	}
	if len(connected) != 1 {
		return "", fmt.Errorf("zernio has %d connected %s accounts, and the job posts to exactly one", len(connected), platform)
	}
	return connected[0], nil
}

func (s *Service) upload(ctx context.Context, name string, video []byte) (string, error) {
	var presigned struct {
		UploadURL string `json:"uploadUrl"`
		PublicURL string `json:"publicUrl"`
	}
	err := s.call(ctx, http.MethodPost, "/media/presign", nil,
		map[string]any{"filename": name, "contentType": videoType, "size": len(video)}, &presigned)
	if err != nil {
		return "", err
	}
	if presigned.UploadURL == "" || presigned.PublicURL == "" {
		return "", errors.New("zernio gave no address to upload the video to")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, presigned.UploadURL, bytes.NewReader(video))
	if err != nil {
		return "", errors.New("zernio gave an address to upload the video to that is not a URL")
	}
	req.Header.Set("Content-Type", videoType)

	res, err := s.http.Do(req)
	if err != nil {
		return "", unreachable(err)
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxBodyBytes))

	if res.StatusCode/100 != 2 {
		return "", refusal(res, "the upload of the video", false)
	}
	return presigned.PublicURL, nil
}

func (s *Service) call(ctx context.Context, method, path string, headers map[string]string, body, answer any) error {
	var sent io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode the request: %w", err)
		}
		sent = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.base+path, sent)
	if err != nil {
		return fmt.Errorf("failed to build the request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.key)
	req.Header.Set("Content-Type", "application/json")
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	res, err := s.http.Do(req)
	if err != nil {
		return unreachable(err)
	}
	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(res.Body, maxBodyBytes))
	if err != nil {
		return unreachable(err)
	}

	var failure struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if res.StatusCode/100 != 2 {
		_ = json.Unmarshal(raw, &failure)
		// A try made while the one before it is still in flight: that one's answer comes after the wait.
		return refusal(res, fmt.Sprintf("%s %s: %s", method, path, failure.Error), failure.Code == "idempotency_conflict")
	}
	if err := json.Unmarshal(raw, answer); err != nil {
		return fmt.Errorf("zernio answered %d to %s %s with a body that is not JSON: %w", res.StatusCode, method, path, err)
	}
	return nil
}

func refusal(res *http.Response, what string, inFlight bool) error {
	reason := fmt.Sprintf("zernio answered %d to %s", res.StatusCode, what)
	if !inFlight && res.StatusCode != http.StatusTooManyRequests && res.StatusCode/100 != 5 {
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
