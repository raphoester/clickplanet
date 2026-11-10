// The server's publishing module, as the job reaches it: over ssh, on the console listener, as fetch asks for a replay.
// The one package of the job that posts.
package ssh_publishing

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
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/posts/metadata"
)

const OpenSSH = "ssh"

// The body comes on the standard input, never the command line: it holds the model's headline and the clip's words,
// which no shell should read. The method is one of this package's constants.
const operatorTool = "cd /opt/clickplanet/deploy/vps && docker compose exec -T backend wget -qO-" +
	" --header 'Content-Type: application/json' --post-file=/dev/stdin" +
	" http://127.0.0.1:8081/publishing.v1.AdminService/"

const maxBodyBytes = 1 << 16

var decoder = protojson.UnmarshalOptions{DiscardUnknown: true}

// host: user@host of the machine the backend runs on. client: the ssh program. httpClient: the upload's, straight to
// the posting service's storage.
func New(host, client string, httpClient *http.Client) (*Service, error) {
	if host == "" {
		return nil, errors.New("the ssh host is empty")
	}
	return &Service{host: host, client: client, http: httpClient, uploaded: map[posts.VideoFile]upload{}}, nil
}

type Service struct {
	host   string
	client string
	http   *http.Client

	mu sync.Mutex
	// Each video once a run, however many times its post is tried.
	uploaded map[posts.VideoFile]upload
}

type upload struct {
	media string
	key   string
}

var (
	_ posts.Planner = (*Service)(nil)
	_ posts.Service = (*Service)(nil)
)

// How many more posts each platform takes now.
func (s *Service) Room(ctx context.Context, platforms []posts.Platform) (map[posts.Platform]int, error) {
	var answer publishingv1.GetRoomResponse
	if err := s.ask(ctx, "GetRoom", &publishingv1.GetRoomRequest{Platforms: wireOf(platforms)}, &answer); err != nil {
		return nil, err
	}
	room := map[posts.Platform]int{}
	for _, platform := range answer.GetRooms() {
		room[platformOf(platform.GetPlatform())] = int(platform.GetLeft())
	}
	return room, nil
}

// The posts with a story key, on every platform: a post made by hand has none.
func (s *Service) Told(ctx context.Context, since time.Time) ([]posts.Telling, error) {
	var answer publishingv1.ListPostsResponse
	if err := s.ask(ctx, "ListPosts", &publishingv1.ListPostsRequest{Since: timestamppb.New(since)}, &answer); err != nil {
		return nil, err
	}
	told := make([]posts.Telling, 0, len(answer.GetPosts()))
	for _, post := range answer.GetPosts() {
		kept := post.GetMetadata().AsMap()
		key, keyed := metadata.KeyOf(kept)
		if !keyed || post.GetScheduledFor() == nil {
			continue
		}
		told = append(told, posts.Telling{
			Key: key, Countries: metadata.CountriesOf(kept), At: post.GetScheduledFor().AsTime(), Title: post.GetTitle(),
			State: post.GetStatus(), Failure: post.GetFailure(),
		})
	}
	return told, nil
}

// The slot the calendar gives a post on the platforms. An ssh that fails is busy: the backend may be restarting.
func (s *Service) Slot(ctx context.Context, platforms []posts.Platform) (posts.Slot, bool, error) {
	request := &publishingv1.PlanPostRequest{Platforms: wireOf(platforms)}
	var answer publishingv1.PlanPostResponse
	if err := s.ask(ctx, "PlanPost", request, &answer); err != nil {
		return posts.Slot{}, false, &posts.BusyError{Reason: err.Error()}
	}
	if answer.GetSlot() == nil {
		return posts.Slot{}, false, nil
	}
	slot := posts.Slot{At: answer.GetSlot().GetAt().AsTime()}
	for _, platform := range answer.GetSlot().GetPlatforms() {
		slot.Platforms = append(slot.Platforms, platformOf(platform))
	}
	return slot, true, nil
}

// The video uploaded once to the service's storage, then the post at its slot. What became of it is the module's
// answer: busy, the slot taken, or failed.
func (s *Service) Schedule(ctx context.Context, post posts.Post) (posts.Scheduled, error) {
	snapshot := post.TakeSnapshot()
	kept, err := metadata.Of(snapshot)
	if err != nil {
		return posts.Scheduled{}, err //nolint:wrapcheck // the record already says what is wrong with it.
	}
	written, err := json.Marshal(kept)
	if err != nil {
		return posts.Scheduled{}, fmt.Errorf("failed to write what the post keeps: %w", err)
	}
	keeps := &structpb.Struct{}
	if err := protojson.Unmarshal(written, keeps); err != nil {
		return posts.Scheduled{}, fmt.Errorf("failed to write what the post keeps: %w", err)
	}

	video, err := s.upload(ctx, snapshot.Video)
	if err != nil {
		return posts.Scheduled{}, err
	}

	var answer publishingv1.SchedulePostResponse
	if err := s.ask(ctx, "SchedulePost", &publishingv1.SchedulePostRequest{
		Slot:     &publishingv1.Slot{At: timestamppb.New(snapshot.At), Platforms: wireOf(snapshot.Platforms)},
		MediaUrl: video.media, IdempotencyKey: video.key, Title: snapshot.Title,
		Tiktok: &publishingv1.TikTokWords{Caption: snapshot.TikTok.Caption},
		Youtube: &publishingv1.YouTubeWords{
			Title: snapshot.YouTube.Title, Description: snapshot.YouTube.Description, Tags: snapshot.YouTube.Tags,
		},
		Metadata: keeps,
	}, &answer); err != nil {
		return posts.Scheduled{}, &posts.BusyError{Reason: err.Error()}
	}
	return outcomeOf(&answer)
}

func outcomeOf(answer *publishingv1.SchedulePostResponse) (posts.Scheduled, error) {
	switch outcome := answer.GetOutcome().(type) {
	case *publishingv1.SchedulePostResponse_Scheduled:
		scheduled := posts.Scheduled{ID: posts.PostID(outcome.Scheduled.GetPostId()), Refused: map[posts.Platform]error{}}
		for _, refusal := range outcome.Scheduled.GetRefusals() {
			scheduled.Refused[platformOf(refusal.GetPlatform())] = errors.New(refusal.GetReason())
		}
		return scheduled, nil
	case *publishingv1.SchedulePostResponse_Busy:
		return posts.Scheduled{}, &posts.BusyError{RetryAfter: outcome.Busy.GetRetryAfter().AsDuration(), Reason: outcome.Busy.GetReason()}
	case *publishingv1.SchedulePostResponse_Taken:
		return posts.Scheduled{}, fmt.Errorf("%w: %s", posts.ErrTaken, outcome.Taken.GetReason())
	case *publishingv1.SchedulePostResponse_Failed:
		return posts.Scheduled{}, errors.New(outcome.Failed.GetReason())
	default:
		return posts.Scheduled{}, errors.New("the backend said nothing of the post")
	}
}

// Once a run: the same file is the same media and the same key. The key is the SHA-256 of the video, so a run made
// again is the same post on the service, not a second one.
func (s *Service) upload(ctx context.Context, file posts.VideoFile) (upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if done, sent := s.uploaded[file]; sent {
		return done, nil
	}

	video, err := os.ReadFile(string(file)) //nolint:gosec // the file of a clip this job rendered.
	if err != nil {
		return upload{}, fmt.Errorf("failed to read the video: %w", err)
	}
	var link publishingv1.CreateMediaUploadResponse
	if err := s.ask(ctx, "CreateMediaUpload", &publishingv1.CreateMediaUploadRequest{
		FileName: filepath.Base(string(file)), ContentType: "video/mp4", SizeBytes: uint64(len(video)),
	}, &link); err != nil {
		return upload{}, &posts.BusyError{Reason: err.Error()}
	}
	if err := s.put(ctx, link.GetUploadUrl(), video); err != nil {
		return upload{}, err
	}

	key := sha256.Sum256(video)
	done := upload{media: link.GetMediaUrl(), key: hex.EncodeToString(key[:])}
	s.uploaded[file] = done
	return done, nil
}

// Straight to the storage, which takes no key: the address is a credential for an hour, so no error says it.
func (s *Service) put(ctx context.Context, address string, video []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, address, bytes.NewReader(video))
	if err != nil {
		return errors.New("the backend gave an address to upload the video to that is not a URL")
	}
	req.Header.Set("Content-Type", "video/mp4")

	res, err := s.http.Do(req)
	if err != nil {
		var failed *url.Error
		if errors.As(err, &failed) {
			err = failed.Err
		}
		return &posts.BusyError{Reason: fmt.Sprintf("the storage could not be reached: %s", err)}
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxBodyBytes))

	reason := fmt.Sprintf("the storage answered %d to the upload of the video", res.StatusCode)
	switch {
	case res.StatusCode/100 == 2:
		return nil
	case res.StatusCode == http.StatusTooManyRequests || res.StatusCode/100 == 5:
		return &posts.BusyError{Reason: reason}
	default:
		return errors.New(reason)
	}
}

func (s *Service) ask(ctx context.Context, method string, request, answer proto.Message) error {
	body, err := protojson.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to write the request of %s: %w", method, err)
	}

	var answered, complained bytes.Buffer
	// "--" ends the options: a host that starts with a dash is a host, never an option of ssh.
	call := exec.CommandContext(ctx, s.client, "--", s.host, operatorTool+method) //nolint:gosec // the operator of the job names the host.
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

func wireOf(platforms []posts.Platform) []publishingv1.Platform {
	wire := make([]publishingv1.Platform, 0, len(platforms))
	for _, platform := range platforms {
		switch platform {
		case posts.TikTok:
			wire = append(wire, publishingv1.Platform_PLATFORM_TIKTOK)
		case posts.YouTube:
			wire = append(wire, publishingv1.Platform_PLATFORM_YOUTUBE)
		default:
			wire = append(wire, publishingv1.Platform_PLATFORM_UNSPECIFIED)
		}
	}
	return wire
}

func platformOf(platform publishingv1.Platform) posts.Platform {
	switch platform {
	case publishingv1.Platform_PLATFORM_TIKTOK:
		return posts.TikTok
	case publishingv1.Platform_PLATFORM_YOUTUBE:
		return posts.YouTube
	default:
		return posts.Platform(strings.ToLower(strings.TrimPrefix(platform.String(), "PLATFORM_")))
	}
}
