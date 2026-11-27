// Package reviews warns the staff, in a private channel, of a flag that waits on a person: its player asked a person to
// look again at the judge's rejection, or the judge was unsure and it is paid.
package reviews

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
)

var ErrUnknownCause = errors.New("no such cause")

// The flags module's id of a submission.
type SubmissionID string

type Cause string

const (
	Appeal Cause = "appeal"
	Unsure Cause = "unsure"
)

// Review is a flag that waits on a person, told once by its submission and its cause.
type Review struct {
	Submission SubmissionID
	Name       string
	Cause      Cause
	// What decided the judge, in its words.
	Why     string
	AskedAt time.Time
}

type Poster interface {
	Post(ctx context.Context, channel discord.ChannelID, text string, audience discord.Audience) error
}

func NewDesk(poster Poster, channel discord.ChannelID, backofficeURL string) Desk {
	return Desk{poster: poster, channel: channel, backofficeURL: strings.TrimSuffix(backofficeURL, "/")}
}

// Desk posts to the staff's channel, with a link to the submission on the backoffice. It pings nobody: the channel's
// readers see it.
type Desk struct {
	poster        Poster
	channel       discord.ChannelID
	backofficeURL string
}

func (d Desk) Tell(ctx context.Context, review Review) error {
	text, err := d.textOf(review)
	if err != nil {
		return err
	}
	if err := d.poster.Post(ctx, d.channel, text, discord.Nobody()); err != nil {
		return fmt.Errorf("failed to tell the staff of %s: %w", review.Submission, err)
	}
	return nil
}

func (d Desk) textOf(review Review) (string, error) {
	link := d.backofficeURL + "/#/submissions/" + string(review.Submission)
	switch review.Cause {
	case Appeal:
		return fmt.Sprintf("🙋 **%s**: its player asks a person to look again at the judge's rejection. Judge: %s %s",
			escaped(review.Name), quoted(review.Why), link), nil
	case Unsure:
		return fmt.Sprintf("🤔 **%s** is paid and waits on you: the judge was unsure. Judge: %s %s",
			escaped(review.Name), quoted(review.Why), link), nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownCause, review.Cause)
}

// A judge's why is a few words: a longer one is cut.
const maxWhy = 300

func quoted(why string) string {
	if utf8.RuneCountInString(why) > maxWhy {
		why = string([]rune(why)[:maxWhy]) + "…"
	}
	if why == "" {
		return "no reason given."
	}
	return "“" + escaped(why) + "”"
}

// A player's name, and words a model wrote about it, are text: none of Discord's markdown.
var markdown = strings.NewReplacer(
	`\`, `\\`, "*", `\*`, "_", `\_`, "~", `\~`, "`", "\\`", "|", `\|`, ">", `\>`, "#", `\#`, "[", `\[`, "]", `\]`,
	"(", `\(`, ")", `\)`, "<", `\<`, "\n", " ", "\r", " ",
)

func escaped(text string) string {
	return markdown.Replace(text)
}
