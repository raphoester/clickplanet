package cpmail

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

const (
	DeliveryCloudflare = "cloudflare"
	DeliveryLog        = "log"
)

type Config struct {
	Delivery string

	From     string
	FromName string

	Cloudflare CloudflareConfig
}

func (c Config) Sender() Named {
	return Named{Address: Address(c.From), Name: c.FromName}
}

func (c Config) Validate() error {
	var errs []error
	if _, err := AddressOf(c.From); err != nil {
		errs = append(errs, fmt.Errorf("from %q: %w", c.From, err))
	}
	switch c.Delivery {
	case DeliveryLog:
	case DeliveryCloudflare:
		if c.Cloudflare.AccountID == "" {
			errs = append(errs, errors.New("cloudflare.accountId is empty while delivery is cloudflare"))
		}
		if c.Cloudflare.APIToken == "" {
			errs = append(errs, errors.New("cloudflare.apiToken is empty while delivery is cloudflare"))
		}
	default:
		errs = append(errs, fmt.Errorf("delivery %q is neither %q nor %q", c.Delivery, DeliveryCloudflare, DeliveryLog))
	}
	return errors.Join(errs...)
}

func New(config Config, logger *slog.Logger, httpClient *http.Client) (Mailer, error) {
	if config.Delivery != DeliveryCloudflare {
		return NewLog(logger), nil
	}
	mailer, err := NewCloudflare(config.Cloudflare, CloudflareAPI, httpClient)
	if err != nil {
		return nil, err
	}
	return mailer, nil
}
