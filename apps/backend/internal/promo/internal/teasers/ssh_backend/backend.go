// The server as the teasers job reaches it: over ssh, on the console listener, as the clips job does. Promo's own
// ListTeasers, and the publishing module that plans and makes the posts. The one package of the job that posts.
package ssh_backend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teasers"
)

const OpenSSH = "ssh"

// The body comes on the standard input, never the command line: it holds the operator's words of the cards, which no
// shell should read. The service and the method are this package's constants.
const operatorTool = "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO-" +
	" --header 'Content-Type: application/json' --post-file=/dev/stdin http://127.0.0.1:8081/"

const (
	promoService      = "promo.v1.AdminService/"
	publishingService = "publishing.v1.AdminService/"
	maxBodyBytes      = 1 << 16
)

var decoder = protojson.UnmarshalOptions{DiscardUnknown: true}

// host: user@host of the machine the backend runs on. client: the ssh program. httpClient: the uploads', straight to
// the posting service's storage.
func New(host, client string, httpClient *http.Client) (*Backend, error) {
	if host == "" {
		return nil, errors.New("the ssh host is empty")
	}
	return &Backend{host: host, client: client, http: httpClient, uploaded: map[teasers.File]string{}}, nil
}

type Backend struct {
	host   string
	client string
	http   *http.Client

	mu sync.Mutex
	// Each file once a run, however many times its post is tried: the address the post names it by.
	uploaded map[teasers.File]string
}

var _ teasers.Publishing = (*Backend)(nil)

// The battles starting within the window, with their cards, by start.
func (b *Backend) Teasers(ctx context.Context, within time.Duration) ([]teasers.Teaser, error) {
	var answer promov1.ListTeasersResponse
	if err := b.ask(ctx, promoService+"ListTeasers", &promov1.ListTeasersRequest{Within: durationpb.New(within)}, &answer); err != nil {
		return nil, err
	}
	listed := make([]teasers.Teaser, 0, len(answer.GetTeasers()))
	for _, teaser := range answer.GetTeasers() {
		listed = append(listed, teaserOf(teaser))
	}
	return listed, nil
}

// The battles the posts since then tease, whatever became of each post: one cancelled in the dashboard was cancelled on
// purpose.
func (b *Backend) Teased(ctx context.Context, since time.Time) ([]teasers.BattleID, error) {
	var answer publishingv1.ListPostsResponse
	if err := b.ask(ctx, publishingService+"ListPosts", &publishingv1.ListPostsRequest{Since: timestamppb.New(since)}, &answer); err != nil {
		return nil, err
	}
	var teased []teasers.BattleID
	for _, post := range answer.GetPosts() {
		battle := post.GetMetadata().GetFields()[teasers.MetadataBattle].GetStringValue()
		if battle != "" && !slices.Contains(teased, teasers.BattleID(battle)) {
			teased = append(teased, teasers.BattleID(battle))
		}
	}
	return teased, nil
}

// The first slot from notBefore to by at which every platform takes the post. An ssh that fails is busy: the backend
// may be restarting.
func (b *Backend) Slot(ctx context.Context, platforms []teasers.Platform, notBefore, by time.Time) (teasers.Slot, bool, error) {
	var answer publishingv1.PlanPostResponse
	if err := b.ask(ctx, publishingService+"PlanPost", &publishingv1.PlanPostRequest{
		Platforms: wireOf(platforms), NotBefore: timestamppb.New(notBefore), By: timestamppb.New(by),
	}, &answer); err != nil {
		return teasers.Slot{}, false, &teasers.BusyError{Reason: err.Error()}
	}
	if answer.GetSlot() == nil {
		return teasers.Slot{}, false, nil
	}
	slot := teasers.Slot{At: answer.GetSlot().GetAt().AsTime()}
	for _, platform := range answer.GetSlot().GetPlatforms() {
		slot.Platforms = append(slot.Platforms, platformOf(platform))
	}
	return slot, true, nil
}

// The files uploaded once to the service's storage (the photos only when TikTok is in the slot), then the post at its
// slot. What became of it is the module's answer: busy, the slot taken, or failed.
func (b *Backend) Schedule(ctx context.Context, post teasers.Post) (teasers.Receipt, error) {
	keeps, err := structpb.NewStruct(post.Metadata())
	if err != nil {
		return teasers.Receipt{}, fmt.Errorf("failed to write what the post keeps: %w", err)
	}
	video, err := b.upload(ctx, post.Media.Video, "video/mp4")
	if err != nil {
		return teasers.Receipt{}, err
	}
	var photos []string
	if slices.Contains(post.Slot.Platforms, teasers.TikTok) {
		for _, file := range post.Media.Photos {
			photo, err := b.upload(ctx, file, "image/jpeg")
			if err != nil {
				return teasers.Receipt{}, err
			}
			photos = append(photos, photo)
		}
	}

	words := post.Teaser.Words()
	var answer publishingv1.SchedulePostResponse
	if err := b.ask(ctx, publishingService+"SchedulePost", &publishingv1.SchedulePostRequest{
		Slot:     &publishingv1.Slot{At: timestamppb.New(post.Slot.At), Platforms: wireOf(post.Slot.Platforms)},
		MediaUrl: video, IdempotencyKey: post.Key(), Title: words.Title,
		Tiktok: &publishingv1.TikTokWords{Caption: words.TikTok.Caption, Title: words.TikTok.Title},
		Youtube: &publishingv1.YouTubeWords{
			Title: words.YouTube.Title, Description: words.YouTube.Description, Tags: words.YouTube.Tags,
		},
		TiktokPhotoUrls: photos,
		Metadata:        keeps,
	}, &answer); err != nil {
		return teasers.Receipt{}, &teasers.BusyError{Reason: err.Error()}
	}
	return receiptOf(&answer)
}

func receiptOf(answer *publishingv1.SchedulePostResponse) (teasers.Receipt, error) {
	switch outcome := answer.GetOutcome().(type) {
	case *publishingv1.SchedulePostResponse_Scheduled:
		receipt := teasers.Receipt{ID: outcome.Scheduled.GetPostId(), Refused: map[teasers.Platform]string{}}
		for _, refusal := range outcome.Scheduled.GetRefusals() {
			receipt.Refused[platformOf(refusal.GetPlatform())] = refusal.GetReason()
		}
		return receipt, nil
	case *publishingv1.SchedulePostResponse_Busy:
		return teasers.Receipt{}, &teasers.BusyError{RetryAfter: outcome.Busy.GetRetryAfter().AsDuration(), Reason: outcome.Busy.GetReason()}
	case *publishingv1.SchedulePostResponse_Taken:
		return teasers.Receipt{}, fmt.Errorf("%w: %s", teasers.ErrTaken, outcome.Taken.GetReason())
	case *publishingv1.SchedulePostResponse_Failed:
		return teasers.Receipt{}, errors.New(outcome.Failed.GetReason())
	default:
		return teasers.Receipt{}, errors.New("the backend said nothing of the post")
	}
}

// Once a run: the same file is the same address.
func (b *Backend) upload(ctx context.Context, file teasers.File, contentType string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if address, sent := b.uploaded[file]; sent {
		return address, nil
	}
	content, err := os.ReadFile(string(file)) //nolint:gosec // a file this job's renderer wrote.
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", filepath.Base(string(file)), err)
	}
	var link publishingv1.CreateMediaUploadResponse
	if err := b.ask(ctx, publishingService+"CreateMediaUpload", &publishingv1.CreateMediaUploadRequest{
		FileName: filepath.Base(string(file)), ContentType: contentType, SizeBytes: uint64(len(content)),
	}, &link); err != nil {
		return "", &teasers.BusyError{Reason: err.Error()}
	}
	if err := b.put(ctx, link.GetUploadUrl(), contentType, content); err != nil {
		return "", err
	}
	b.uploaded[file] = link.GetMediaUrl()
	return link.GetMediaUrl(), nil
}

// Straight to the storage, which takes no key: the address is a credential for an hour, so no error says it.
func (b *Backend) put(ctx context.Context, address, contentType string, content []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, address, bytes.NewReader(content))
	if err != nil {
		return errors.New("the backend gave an address to upload a file to that is not a URL")
	}
	req.Header.Set("Content-Type", contentType)
	res, err := b.http.Do(req)
	if err != nil {
		var failed *url.Error
		if errors.As(err, &failed) {
			err = failed.Err
		}
		return &teasers.BusyError{Reason: fmt.Sprintf("the storage could not be reached: %s", err)}
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxBodyBytes))

	reason := fmt.Sprintf("the storage answered %d to the upload of a file", res.StatusCode)
	switch {
	case res.StatusCode/100 == 2:
		return nil
	case res.StatusCode == http.StatusTooManyRequests || res.StatusCode/100 == 5:
		return &teasers.BusyError{Reason: reason}
	default:
		return errors.New(reason)
	}
}

func (b *Backend) ask(ctx context.Context, method string, request, answer proto.Message) error {
	body, err := protojson.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to write the request of %s: %w", method, err)
	}
	var answered, complained bytes.Buffer
	// "--" ends the options: a host that starts with a dash is a host, never an option of ssh.
	call := exec.CommandContext(ctx, b.client, "--", b.host, operatorTool+method) //nolint:gosec // the operator of the job names the host.
	call.Stdin = bytes.NewReader(body)
	call.Stdout = &answered
	call.Stderr = &complained
	if err := call.Run(); err != nil {
		return fmt.Errorf("ssh failed to ask %s: %w: %s", method, err, strings.TrimSpace(complained.String()))
	}
	if err := decoder.Unmarshal(answered.Bytes(), answer); err != nil {
		return fmt.Errorf("the backend answered %s with what is not a %s: %w", method, answer.ProtoReflect().Descriptor().FullName(), err)
	}
	return nil
}

// The domain's teaser of promo's.
func teaserOf(wire *promov1.Teaser) teasers.Teaser {
	teaser := teasers.Teaser{
		Battle: teasers.BattleID(wire.GetBattleId()), StartsAt: wire.GetStartsAt().AsTime(), EndsAt: wire.GetEndsAt().AsTime(),
		Description: wire.GetDescription(), Times: wire.GetTimes(), EndsSeason: wire.GetEndsSeason(),
	}
	for _, card := range wire.GetCards() {
		teaser.Cards = append(teaser.Cards, teasers.Card{
			BackgroundURL: card.GetBackgroundUrl(), Headline: card.GetHeadline(), Line: card.GetLine(),
		})
	}
	return teaser
}

func wireOf(platforms []teasers.Platform) []publishingv1.Platform {
	wire := make([]publishingv1.Platform, 0, len(platforms))
	for _, platform := range platforms {
		switch platform {
		case teasers.TikTok:
			wire = append(wire, publishingv1.Platform_PLATFORM_TIKTOK)
		case teasers.YouTube:
			wire = append(wire, publishingv1.Platform_PLATFORM_YOUTUBE)
		default:
			wire = append(wire, publishingv1.Platform_PLATFORM_UNSPECIFIED)
		}
	}
	return wire
}

func platformOf(platform publishingv1.Platform) teasers.Platform {
	switch platform {
	case publishingv1.Platform_PLATFORM_TIKTOK:
		return teasers.TikTok
	case publishingv1.Platform_PLATFORM_YOUTUBE:
		return teasers.YouTube
	default:
		return teasers.Platform(strings.ToLower(strings.TrimPrefix(platform.String(), "PLATFORM_")))
	}
}
