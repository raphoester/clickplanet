package s3_attachment_bucket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var ErrNoBucket = errors.New("no bucket is set for the attachments")

const (
	readTTL     = time.Minute
	readsAtOnce = 8
)

type Config struct {
	Endpoint        string
	Region          string
	Name            string
	AccessKeyID     string
	SecretAccessKey string
}

func (c Config) String() string {
	return fmt.Sprintf("{Endpoint:%s Region:%s Name:%s AccessKeyID:%s}", c.Endpoint, c.Region, c.Name, c.AccessKeyID)
}

func (c Config) Validate() error {
	var errs []error
	if _, err := originOf(c.Endpoint); err != nil {
		errs = append(errs, fmt.Errorf("endpoint %q: %w", c.Endpoint, err))
	}
	for name, value := range map[string]string{
		"region": c.Region, "name": c.Name, "accessKeyId": c.AccessKeyID, "secretAccessKey": c.SecretAccessKey,
	} {
		if value == "" {
			errs = append(errs, fmt.Errorf("%s is empty", name))
		}
	}
	return errors.Join(errs...)
}

func originOf(endpoint string) (*url.URL, error) {
	origin, err := url.Parse(endpoint)
	switch {
	case err != nil:
		return nil, fmt.Errorf("not a URL: %w", err)
	case origin.Scheme != "http" && origin.Scheme != "https", origin.Host == "":
		return nil, errors.New("not an http or https URL")
	case origin.Path != "" && origin.Path != "/", origin.RawQuery != "", origin.Fragment != "", origin.User != nil:
		return nil, errors.New("more than a scheme and a host")
	}
	return origin, nil
}

type Lifetimes struct {
	Upload time.Duration
	View   time.Duration
}

// An empty config builds a bucket every link of which is ErrNoBucket: the module that takes no attachment asks for none.
func New(config Config, lifetimes Lifetimes, clock cptime.Clock, httpClient *http.Client) (*Bucket, error) {
	bucket := &Bucket{lifetimes: lifetimes, clock: clock, http: httpClient}
	if config == (Config{}) {
		return bucket, nil
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}
	origin, err := originOf(config.Endpoint)
	if err != nil {
		return nil, err
	}

	bucket.origin = origin.Scheme + "://" + origin.Host
	bucket.host = origin.Host
	bucket.name = config.Name
	bucket.credentials = credentials{keyID: config.AccessKeyID, secret: config.SecretAccessKey, region: config.Region}
	return bucket, nil
}

type Bucket struct {
	origin      string
	host        string
	name        string
	credentials credentials
	lifetimes   Lifetimes
	clock       cptime.Clock
	http        *http.Client
}

func (b *Bucket) UploadURL(file notes.Attachment) (string, error) {
	return b.link(http.MethodPut, file.ID.String(), b.lifetimes.Upload, map[string]string{
		"content-type":   string(file.MediaType),
		"content-length": strconv.FormatInt(file.Size, 10),
	})
}

func (b *Bucket) ViewURL(attachment string) (string, error) {
	return b.link(http.MethodGet, attachment, b.lifetimes.View, nil)
}

// What the bucket holds under each id, in their order: the type and the size the upload link signed.
func (b *Bucket) Attachments(ctx context.Context, ids ...notes.AttachmentID) ([]notes.Attachment, error) {
	files := make([]notes.Attachment, len(ids))
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(readsAtOnce)

	for n, id := range ids {
		group.Go(func() error {
			file, err := b.attachment(ctx, id)
			files[n] = file
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err //nolint:wrapcheck // attachment named what failed.
	}
	return files, nil
}

func (b *Bucket) attachment(ctx context.Context, id notes.AttachmentID) (notes.Attachment, error) {
	link, err := b.link(http.MethodHead, id.String(), readTTL, nil)
	if err != nil {
		return notes.Attachment{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, link, nil)
	if err != nil {
		return notes.Attachment{}, fmt.Errorf("failed to build the request that reads the file %s: %w", id, err)
	}
	res, err := b.http.Do(req)
	if err != nil {
		return notes.Attachment{}, fmt.Errorf("failed to reach the bucket to read the file %s: %w", id, err)
	}
	_ = res.Body.Close()

	switch {
	case res.StatusCode == http.StatusNotFound:
		return notes.Attachment{}, fmt.Errorf("%w: %s", notes.ErrNotUploaded, id)
	case res.StatusCode != http.StatusOK:
		return notes.Attachment{}, fmt.Errorf("the bucket answered %d to the read of the file %s", res.StatusCode, id)
	}

	size, err := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return notes.Attachment{}, fmt.Errorf("the bucket gave the file %s no size: %w", id, err)
	}
	return notes.Attachment{ID: id, MediaType: notes.MediaType(res.Header.Get("Content-Type")), Size: size}, nil
}

func (b *Bucket) link(method string, attachment string, ttl time.Duration, headers map[string]string) (string, error) {
	if b.origin == "" {
		return "", ErrNoBucket
	}

	path := "/" + escaped(b.name) + "/" + escaped(attachment)
	query := b.credentials.signedQuery(request{
		method: method, host: b.host, path: path, headers: headers, at: b.clock.Now(), ttl: ttl,
	})
	return b.origin + path + "?" + query, nil
}
