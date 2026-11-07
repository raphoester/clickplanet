package judgements

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
)

// The slate is wiped after 7 days with no offence: the judge reads that much of an account's history.
const SlateWindow = 7 * 24 * time.Hour

var offenceReason = regexp.MustCompile(`^offence (\d+) \((chat|gifs)\):`)

const firstSpam = "spam, no offence"

// What an account's recent history says of one kind (the chat or the GIFs), from the reasons each action carries: the
// steps are counted from the history, never from a counter of the judge's own.
type Slate struct {
	// The highest offence the window names: "offence 3 (chat): …" was the third, so the next is the fourth. A mute and
	// a removal of one offence name the same number, so it is never counted twice.
	offences int
	// A mute of the kind of 24 hours or more: whoever goes on after one is the operator's.
	dayMuted bool
	// A spam removal that was no offence: spam again is one.
	spammed bool
}

func SlateOf(history []moderation.Snapshot, kind mutes.Kind, since time.Time) Slate {
	var slate Slate
	for _, action := range history {
		if action.At.Before(since) {
			continue
		}
		if action.Kind == moderation.Muted && action.MuteKind == kind && action.Duration >= MaxMute {
			slate.dayMuted = true
		}
		reason := string(action.Reason)
		if strings.HasPrefix(reason, firstSpam) {
			slate.spammed = true
		}
		matched := offenceReason.FindStringSubmatch(reason)
		if matched == nil || matched[2] != kind.String() {
			continue
		}
		if n, err := strconv.Atoi(matched[1]); err == nil && n > slate.offences {
			slate.offences = n
		}
	}
	return slate
}

// No step mutes longer: longer is the operator's call.
const MaxMute = 24 * time.Hour

// What the steps call for, for the next offence.
type Step struct {
	Offence int
	// 0 for no mute.
	Mute time.Duration
	// After a 24-hour mute: the message goes, and the operator decides the rest.
	ForOperator bool
}

// The policy's table: the 1st removes, the 2nd mutes 15 minutes, the 3rd an hour, the 4th and after 24 hours, and after
// a 24-hour mute the operator decides. A severe offence goes straight to the 3rd step, whatever the count below it.
func (s Slate) Next(severe bool) Step {
	n := s.offences + 1
	if severe {
		n = max(n, 3)
	}
	if s.dayMuted || s.offences >= 4 {
		return Step{Offence: n, ForOperator: true}
	}
	switch n {
	case 1:
		return Step{Offence: n}
	case 2:
		return Step{Offence: n, Mute: 15 * time.Minute}
	case 3:
		return Step{Offence: n, Mute: time.Hour}
	default:
		return Step{Offence: n, Mute: MaxMute}
	}
}
