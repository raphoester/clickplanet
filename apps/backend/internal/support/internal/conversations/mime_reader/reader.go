package mime_reader

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/htmlindex"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
)

// What is read of one part: a reply's words are never this long.
const maxPartBytes = 1 << 20

const maxDepth = 5

type Reader struct{}

func (Reader) Read(raw io.Reader) (conversations.Received, error) {
	message, err := mail.ReadMessage(bufio.NewReader(raw))
	if err != nil {
		return conversations.Received{}, fmt.Errorf("%w: %w", conversations.ErrUnreadable, err)
	}

	id, err := cpmail.MessageIDOf(message.Header.Get("Message-ID"))
	if err != nil {
		return conversations.Received{}, fmt.Errorf("%w: its Message-ID: %w", conversations.ErrUnreadable, err)
	}

	words := &found{}
	if err := words.read(message.Header.Get("Content-Type"), message.Header.Get("Content-Transfer-Encoding"),
		message.Body, 0); err != nil {
		return conversations.Received{}, fmt.Errorf("%w: %w", conversations.ErrUnreadable, err)
	}
	if _, err := io.Copy(io.Discard, message.Body); err != nil {
		return conversations.Received{}, fmt.Errorf("%w: %w", conversations.ErrUnreadable, err)
	}

	return conversations.Received{Mail: id, From: sender(message.Header.Get("From")), Text: words.text()}, nil
}

func sender(header string) cpmail.Address {
	parsed, err := (&mail.AddressParser{WordDecoder: decoder}).Parse(header)
	if err != nil {
		return ""
	}
	address, err := cpmail.AddressOf(parsed.Address)
	if err != nil {
		return ""
	}
	return address
}

var decoder = &mime.WordDecoder{CharsetReader: charsetReader}

func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	return inCharset(charset, input), nil
}

// An unknown charset is read as UTF-8, and cleaned after.
func inCharset(charset string, input io.Reader) io.Reader {
	encoding, err := htmlindex.Get(charset)
	if err != nil {
		return input
	}
	return encoding.NewDecoder().Reader(input)
}

type found struct {
	plain, html *string
}

func (f *found) text() string {
	switch {
	case f.plain != nil:
		return *f.plain
	case f.html != nil:
		return stripped(*f.html)
	default:
		return ""
	}
}

func (f *found) read(contentType, transfer string, body io.Reader, depth int) error {
	media, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		media, params = "text/plain", map[string]string{}
	}

	if strings.HasPrefix(media, "multipart/") {
		if depth >= maxDepth {
			return nil
		}
		parts := multipart.NewReader(body, params["boundary"])
		for {
			part, err := parts.NextRawPart()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to read a part: %w", err)
			}
			if attached(part.Header.Get("Content-Disposition")) {
				continue
			}
			if err := f.read(part.Header.Get("Content-Type"), part.Header.Get("Content-Transfer-Encoding"), part, depth+1); err != nil {
				return err
			}
		}
	}

	if media != "text/plain" && media != "text/html" || media == "text/plain" && f.plain != nil || media == "text/html" && f.html != nil {
		return nil
	}

	read, err := io.ReadAll(inCharset(params["charset"], io.LimitReader(decoded(transfer, body), maxPartBytes)))
	if err != nil {
		return fmt.Errorf("failed to decode a %s part: %w", media, err)
	}

	text := string(read)
	if media == "text/plain" {
		f.plain = &text
	} else {
		f.html = &text
	}
	return nil
}

func attached(disposition string) bool {
	kind, _, err := mime.ParseMediaType(disposition)
	return err == nil && kind == "attachment"
}

func decoded(transfer string, body io.Reader) io.Reader {
	switch strings.ToLower(strings.TrimSpace(transfer)) {
	case "quoted-printable":
		return quotedprintable.NewReader(body)
	case "base64":
		return base64.NewDecoder(base64.StdEncoding, &unwrapped{r: body})
	default:
		return body
	}
}

// base64 in a mail is cut in lines, which the decoder does not skip.
type unwrapped struct {
	r io.Reader
}

func (u *unwrapped) Read(p []byte) (int, error) {
	n, err := u.r.Read(p)
	kept := 0
	for _, b := range p[:n] {
		if b != '\r' && b != '\n' && b != ' ' && b != '\t' {
			p[kept] = b
			kept++
		}
	}
	return kept, err //nolint:wrapcheck // io.EOF is read by identity.
}

var (
	hidden = regexp.MustCompile(`(?is)<(style|script|head)[^>]*>.*?</(style|script|head)>`)
	breaks = regexp.MustCompile(`(?i)<br\s*/?>|</(p|div|li|tr|h[1-6]|blockquote)>`)
	tags   = regexp.MustCompile(`(?s)<[^>]*>`)
	quote  = regexp.MustCompile(`(?is)<blockquote[^>]*>.*</blockquote>`)
	blanks = regexp.MustCompile(`\n[ \t]*\n(\s*\n)+`)
)

func stripped(page string) string {
	text := hidden.ReplaceAllString(page, "")
	text = quote.ReplaceAllString(text, "")
	text = breaks.ReplaceAllString(text, "\n")
	text = tags.ReplaceAllString(text, "")
	text = html.UnescapeString(text)
	return blanks.ReplaceAllString(text, "\n\n")
}
