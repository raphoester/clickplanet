package s3_attachment_bucket

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

var ErrNoBucket = errors.New("no bucket is set for the attachments")

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
func New(config Config, lifetimes Lifetimes, clock cptime.Clock) (*Bucket, error) {
	bucket := &Bucket{lifetimes: lifetimes, clock: clock}
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
}

func (b *Bucket) UploadURLs(note notes.Note) ([]string, error) {
	kept := note.TakeSnapshot()

	links := make([]string, 0, len(kept.Attachments))
	for position, file := range kept.Attachments {
		link, err := b.link(http.MethodPut, uuid.UUID(kept.ID).String(), position, b.lifetimes.Upload, map[string]string{
			"content-type":   string(file.MediaType),
			"content-length": strconv.FormatInt(file.Size, 10),
		})
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

func (b *Bucket) ViewURL(note string, position int) (string, error) {
	return b.link(http.MethodGet, note, position, b.lifetimes.View, nil)
}

func (b *Bucket) link(method string, note string, position int, ttl time.Duration, headers map[string]string) (string, error) {
	if b.origin == "" {
		return "", ErrNoBucket
	}

	path := "/" + escaped(b.name) + "/" + escaped(note) + "/" + strconv.Itoa(position)
	query := b.credentials.signedQuery(request{
		method: method, host: b.host, path: path, headers: headers, at: b.clock.Now(), ttl: ttl,
	})
	return b.origin + path + "?" + query, nil
}
