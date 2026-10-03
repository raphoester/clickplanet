package subscriptions

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

var ErrAddressInvalid = errors.New("this is not an email address")

// RFC 5321's limit on a path.
const maxAddressLength = 254

var hostname = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)+[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

type Address string

func AddressOf(raw string) (Address, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) > maxAddressLength {
		return "", fmt.Errorf("%w: it is %d bytes, over %d", ErrAddressInvalid, len(trimmed), maxAddressLength)
	}

	parsed, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrAddressInvalid, err)
	}
	if parsed.Name != "" || parsed.Address != trimmed {
		return "", fmt.Errorf("%w: it is more than a bare address", ErrAddressInvalid)
	}

	address := Address(strings.ToLower(trimmed))
	if !hostname.MatchString(address.domain()) {
		return "", fmt.Errorf("%w: %q is not a domain mail is sent to", ErrAddressInvalid, address.domain())
	}
	return address, nil
}

func (a Address) domain() string {
	return string(a)[strings.LastIndex(string(a), "@")+1:]
}
