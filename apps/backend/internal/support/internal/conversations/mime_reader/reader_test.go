package mime_reader_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/mime_reader"
)

func read(t *testing.T, raw string) conversations.Received {
	t.Helper()

	received, err := mime_reader.Reader{}.Read(strings.NewReader(strings.ReplaceAll(raw, "\n", "\r\n")))
	require.NoError(t, err)
	return received
}

func TestAPlainMailIsItsIDItsSenderAndItsWords(t *testing.T) {
	received := read(t, `From: "Ada Lovelace" <Ada@Example.com>
To: support+019a6f2e-1c2d-7e3f-8a9b-0c1d2e3f4a5b@clickplanet.lol
Subject: Re: your bug report
Message-ID: <CAF1=x@mail.gmail.com>
Content-Type: text/plain; charset="UTF-8"

An iPhone 13 🎉
`)

	assert.Equal(t, conversations.Received{
		Mail: "CAF1=x@mail.gmail.com", From: "ada@example.com", Text: "An iPhone 13 🎉\r\n",
	}, received)
}

func TestTheTextPartIsReadOverTheHTMLAndTheFilesAreLeftOut(t *testing.T) {
	received := read(t, `From: ada@example.com
Message-ID: <1@example.com>
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="outer"

--outer
Content-Type: multipart/alternative; boundary="inner"

--inner
Content-Type: text/plain; charset="iso-8859-1"
Content-Transfer-Encoding: quoted-printable

J'ai un t=E9l=E9phone.
--inner
Content-Type: text/html; charset="UTF-8"

<p>J'ai un t&eacute;l&eacute;phone.</p>
--inner--
--outer
Content-Type: text/plain
Content-Disposition: attachment; filename="log.txt"

not the words
--outer--
`)

	assert.Equal(t, "J'ai un téléphone.", strings.TrimSpace(received.Text))
}

func TestAnHTMLMailAloneIsReadWithNoTagsNoStyleAndNoQuote(t *testing.T) {
	received := read(t, `From: ada@example.com
Message-ID: <2@example.com>
Content-Type: text/html; charset="UTF-8"
Content-Transfer-Encoding: base64

`+wrapped("PGh0bWw+PGhlYWQ+PHN0eWxlPnAge2NvbG9yOnJlZH08L3N0eWxlPjwvaGVhZD48Ym9keT48ZGl2PkFuIGlQaG9uZSAxMy48YnI+U2FmYXJpICZhbXA7IENocm9tZS48L2Rpdj48YmxvY2txdW90ZT5XaGljaCBwaG9uZT88L2Jsb2NrcXVvdGU+PC9ib2R5PjwvaHRtbD4="))

	assert.Equal(t, "An iPhone 13.\nSafari & Chrome.", strings.TrimSpace(received.Text))
}

func wrapped(encoded string) string {
	var lines []string
	for len(encoded) > 40 {
		lines = append(lines, encoded[:40])
		encoded = encoded[40:]
	}
	return strings.Join(append(lines, encoded), "\n") + "\n"
}

func TestAnEncodedSenderNameIsReadAndAFromNoMailGoesBackToIsNone(t *testing.T) {
	named := read(t, "From: =?UTF-8?Q?Z=C3=B6e?= <zoe@example.org>\nMessage-ID: <3@example.com>\n\nHi\n")
	assert.Equal(t, cpmail.Address("zoe@example.org"), named.From)

	nobody := read(t, "From: root@localhost\nMessage-ID: <4@example.com>\n\nHi\n")
	assert.Empty(t, nobody.From)
}

func TestAMailWithNoMessageIDOrNoHeaderCannotBeRead(t *testing.T) {
	for name, raw := range map[string]string{
		"no message id": "From: ada@example.com\r\n\r\nHi\r\n",
		"not a mail":    "Hi",
	} {
		_, err := mime_reader.Reader{}.Read(strings.NewReader(raw))

		require.ErrorIs(t, err, conversations.ErrUnreadable, name)
	}
}
