// The server as the teasers job reaches it: over ssh, on the console listener, as the clips job does. Promo's teasers
// (ListDueTeasers, PostTeaser), and the upload links of the publishing module's storage.
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
	"strings"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	promov1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	publishingv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job"
)

const OpenSSH = "ssh"

// The body comes on the standard input, never the command line. The service and the method are this package's
// constants.
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
	return &Backend{host: host, client: client, http: httpClient, uploaded: map[teaser_job.File]teaser_job.MediaURL{}}, nil
}

type Backend struct {
	host   string
	client string
	http   *http.Client

	mu sync.Mutex
	// Each file once a run, however many times its post is tried.
	uploaded map[teaser_job.File]teaser_job.MediaURL
}

// The battles promo says are due, each with its teaser as promo wrote it.
func (b *Backend) Due(ctx context.Context) ([]teaser_job.Due, error) {
	var answer promov1.ListDueTeasersResponse
	if err := b.ask(ctx, promoService+"ListDueTeasers", &promov1.ListDueTeasersRequest{}, &answer); err != nil {
		return nil, err
	}
	due := make([]teaser_job.Due, 0, len(answer.GetTeasers()))
	for _, teaser := range answer.GetTeasers() {
		written, err := protojson.MarshalOptions{Multiline: true}.Marshal(teaser)
		if err != nil {
			return nil, fmt.Errorf("failed to write the teaser of %s: %w", teaser.GetBattleId(), err)
		}
		due = append(due, teaser_job.Due{
			Battle: teaser_job.BattleID(teaser.GetBattleId()), StartsAt: teaser.GetStartsAt().AsTime(),
			Cards: len(teaser.GetCards()), Teaser: written,
		})
	}
	return due, nil
}

// Once a run: the same file is the same address.
func (b *Backend) Upload(ctx context.Context, file teaser_job.File, contentType string) (teaser_job.MediaURL, error) {
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
		return "", err
	}
	if err := b.put(ctx, link.GetUploadUrl(), contentType, content); err != nil {
		return "", err
	}
	b.uploaded[file] = teaser_job.MediaURL(link.GetMediaUrl())
	return teaser_job.MediaURL(link.GetMediaUrl()), nil
}

// Promo's answer: posted, held, skipped or failed. Busy is a *teaser_job.BusyError, to wait for.
func (b *Backend) Post(ctx context.Context, battle teaser_job.BattleID, media teaser_job.Media) (teaser_job.Outcome, error) {
	request := &promov1.PostTeaserRequest{BattleId: string(battle), VideoUrl: string(media.Video)}
	for _, photo := range media.Photos {
		request.PhotoUrls = append(request.PhotoUrls, string(photo))
	}
	var answer promov1.PostTeaserResponse
	if err := b.ask(ctx, promoService+"PostTeaser", request, &answer); err != nil {
		return teaser_job.Outcome{}, err
	}
	outcome := teaser_job.Outcome{Battle: battle}
	switch said := answer.GetOutcome().(type) {
	case *promov1.PostTeaserResponse_Posted:
		outcome.Result, outcome.Title, outcome.Post = teaser_job.Posted, said.Posted.GetTitle(), said.Posted.GetPostId()
		outcome.At, outcome.Platforms = said.Posted.GetAt().AsTime(), said.Posted.GetPlatforms()
		for _, refusal := range said.Posted.GetRefusals() {
			if outcome.Refused == nil {
				outcome.Refused = map[string]string{}
			}
			outcome.Refused[refusal.GetPlatform()] = refusal.GetReason()
		}
	case *promov1.PostTeaserResponse_Held:
		outcome.Result, outcome.Why = teaser_job.Held, said.Held.GetReason()
	case *promov1.PostTeaserResponse_Skipped:
		outcome.Result, outcome.Why = teaser_job.Skipped, said.Skipped.GetReason()
	case *promov1.PostTeaserResponse_Failed:
		outcome.Result, outcome.Why = teaser_job.Failed, said.Failed.GetReason()
	case *promov1.PostTeaserResponse_Busy:
		return teaser_job.Outcome{}, &teaser_job.BusyError{RetryAfter: said.Busy.GetRetryAfter().AsDuration(), Reason: said.Busy.GetReason()}
	default:
		return teaser_job.Outcome{}, errors.New("promo said nothing of the teaser")
	}
	return outcome, nil
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
		return &teaser_job.BusyError{Reason: fmt.Sprintf("the storage could not be reached: %s", err)}
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxBodyBytes))

	reason := fmt.Sprintf("the storage answered %d to the upload of a file", res.StatusCode)
	switch {
	case res.StatusCode/100 == 2:
		return nil
	case res.StatusCode == http.StatusTooManyRequests || res.StatusCode/100 == 5:
		return &teaser_job.BusyError{Reason: reason}
	default:
		return errors.New(reason)
	}
}

// A call the server refused (busybox's wget says "server returned error: HTTP/1.1 4…") is an error that says why; a
// call that could not be made, or that the server failed, is busy: the backend may be restarting.
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
		complaint := strings.TrimSpace(complained.String())
		if strings.Contains(complaint, "HTTP/1.1 4") {
			return fmt.Errorf("the backend refused %s: %s", method, complaint)
		}
		return &teaser_job.BusyError{Reason: fmt.Sprintf("ssh failed to ask %s: %s: %s", method, err, complaint)}
	}
	if err := decoder.Unmarshal(answered.Bytes(), answer); err != nil {
		return fmt.Errorf("the backend answered %s with what is not a %s: %w", method, answer.ProtoReflect().Descriptor().FullName(), err)
	}
	return nil
}
