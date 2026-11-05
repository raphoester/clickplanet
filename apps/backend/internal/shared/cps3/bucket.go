package cps3

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ErrNoBucket = errors.New("no bucket is set")
	ErrNotFound = errors.New("the bucket holds no such object")
)

const readTTL = time.Minute

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

// An empty config builds a bucket every link of which is ErrNoBucket: the module that keeps no file asks for none.
func New(config Config, clock cptime.Clock, httpClient *http.Client) (*Bucket, error) {
	bucket := &Bucket{clock: clock, http: httpClient}
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
	clock       cptime.Clock
	http        *http.Client
}

type Object struct {
	ContentType string
	Size        int64
}

// A PUT of exactly this type and this size: the caller of the link cannot send anything else.
func (b *Bucket) UploadLink(key string, object Object, ttl time.Duration) (string, error) {
	return b.link(http.MethodPut, key, ttl, map[string]string{
		"content-type":   object.ContentType,
		"content-length": strconv.FormatInt(object.Size, 10),
	}, nil)
}

func (b *Bucket) ViewLink(key string, ttl time.Duration) (string, error) {
	return b.link(http.MethodGet, key, ttl, nil, nil)
}

// A GET the browser saves under the file name instead of showing.
func (b *Bucket) DownloadLink(key string, fileName string, ttl time.Duration) (string, error) {
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": fileName})
	if disposition == "" {
		return "", fmt.Errorf("%q cannot be a file name", fileName)
	}
	return b.link(http.MethodGet, key, ttl, nil, map[string]string{"response-content-disposition": disposition})
}

// What the bucket holds under the key: the type and the size the upload link signed.
func (b *Bucket) Head(ctx context.Context, key string) (Object, error) {
	status, header, err := b.do(ctx, http.MethodHead, key)
	if err != nil {
		return Object{}, err
	}

	switch {
	case status == http.StatusNotFound:
		return Object{}, fmt.Errorf("%w: %s", ErrNotFound, key)
	case status != http.StatusOK:
		return Object{}, fmt.Errorf("the bucket answered %d to the read of %s", status, key)
	}

	size, err := strconv.ParseInt(header.Get("Content-Length"), 10, 64)
	if err != nil {
		return Object{}, fmt.Errorf("the bucket gave %s no size: %w", key, err)
	}
	return Object{ContentType: header.Get("Content-Type"), Size: size}, nil
}

// An object the bucket does not hold is deleted already.
func (b *Bucket) Delete(ctx context.Context, key string) error {
	status, _, err := b.do(ctx, http.MethodDelete, key)
	if err != nil {
		return err
	}

	switch status {
	case http.StatusOK, http.StatusNoContent, http.StatusNotFound:
		return nil
	default:
		return fmt.Errorf("the bucket answered %d to the deletion of %s", status, key)
	}
}

// The status and the headers of a call with no body either way.
func (b *Bucket) do(ctx context.Context, method string, key string) (int, http.Header, error) {
	link, err := b.link(method, key, readTTL, nil, nil)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, link, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to build the %s of %s: %w", method, key, err)
	}
	res, err := b.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to reach the bucket for the %s of %s: %w", method, key, err)
	}
	_ = res.Body.Close()
	return res.StatusCode, res.Header, nil
}

func (b *Bucket) link(method string, key string, ttl time.Duration, headers, query map[string]string) (string, error) {
	if b.origin == "" {
		return "", ErrNoBucket
	}

	path := "/" + escaped(b.name) + "/" + escapedPath(key)
	signed := b.credentials.signedQuery(request{
		method: method, host: b.host, path: path, headers: headers, query: query, at: b.clock.Now(), ttl: ttl,
	})
	return b.origin + path + "?" + signed, nil
}
