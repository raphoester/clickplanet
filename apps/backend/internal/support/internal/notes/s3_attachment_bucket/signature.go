package s3_attachment_bucket

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	algorithm    = "AWS4-HMAC-SHA256"
	service      = "s3"
	terminator   = "aws4_request"
	unsignedBody = "UNSIGNED-PAYLOAD"
	stampLayout  = "20060102T150405Z"
	dayLayout    = "20060102"
)

type credentials struct {
	keyID  string
	secret string
	region string
}

// A request whose link carries its own signature: AWS Signature Version 4, in the query.
type request struct {
	method string
	host   string
	// Already escaped.
	path string
	// Lower-case names. The caller of the link sends each one as it is here, or the bucket refuses it.
	headers map[string]string
	at      time.Time
	ttl     time.Duration
}

func (c credentials) signedQuery(r request) string {
	at := r.at.UTC()
	day := at.Format(dayLayout)
	scope := strings.Join([]string{day, c.region, service, terminator}, "/")

	headers := map[string]string{"host": r.host}
	maps.Copy(headers, r.headers)
	names := slices.Sorted(maps.Keys(headers))

	var sent strings.Builder
	for _, name := range names {
		sent.WriteString(name + ":" + strings.TrimSpace(headers[name]) + "\n")
	}
	signed := strings.Join(names, ";")

	// The bucket reads the parameters sorted by name, and these already are.
	query := "X-Amz-Algorithm=" + algorithm +
		"&X-Amz-Credential=" + escaped(c.keyID+"/"+scope) +
		"&X-Amz-Date=" + at.Format(stampLayout) +
		"&X-Amz-Expires=" + strconv.Itoa(int(r.ttl/time.Second)) +
		"&X-Amz-SignedHeaders=" + escaped(signed)

	canonical := strings.Join([]string{r.method, r.path, query, sent.String(), signed, unsignedBody}, "\n")
	digest := sha256.Sum256([]byte(canonical))
	toSign := strings.Join([]string{algorithm, at.Format(stampLayout), scope, hex.EncodeToString(digest[:])}, "\n")

	key := []byte("AWS4" + c.secret)
	for _, part := range []string{day, c.region, service, terminator} {
		key = mac(key, part)
	}
	return query + "&X-Amz-Signature=" + hex.EncodeToString(mac(key, toSign))
}

func mac(key []byte, message string) []byte {
	hash := hmac.New(sha256.New, key)
	hash.Write([]byte(message))
	return hash.Sum(nil)
}

// RFC 3986, as the signature wants it: every byte but a letter, a digit, - . _ and ~.
func escaped(value string) string {
	var out strings.Builder
	for _, b := range []byte(value) {
		switch {
		case b >= 'A' && b <= 'Z', b >= 'a' && b <= 'z', b >= '0' && b <= '9', strings.IndexByte("-._~", b) >= 0:
			out.WriteByte(b)
		default:
			fmt.Fprintf(&out, "%%%02X", b)
		}
	}
	return out.String()
}
