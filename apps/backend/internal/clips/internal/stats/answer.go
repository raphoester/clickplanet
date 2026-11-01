package stats

import (
	"encoding/json"
	"strconv"
	"time"
)

// What the service answered to the analytics of one post: a 200, a 202 while a platform still syncs, a 424 when every
// platform failed. The others are errors.
type Answer struct {
	HTTPStatus int    `json:"httpStatus"`
	Status     string `json:"status,omitempty"`
	SyncStatus string `json:"syncStatus,omitempty"`
	Message    string `json:"message,omitempty"`

	// Each platform's entry as the service wrote it, metrics and all.
	Entries []Entry `json:"-"`
}

type Entry struct {
	Platform  string
	AccountID string
	Raw       json.RawMessage
}

func (a Answer) entryOf(target Target) json.RawMessage {
	for _, entry := range a.Entries {
		if entry.Platform == target.Platform && (entry.AccountID == "" || target.AccountID == "" || entry.AccountID == target.AccountID) {
			return entry.Raw
		}
	}
	return nil
}

// What YouTube answered to the retention curve of one video, as it came: the curve, or why there is none (a 412 asks
// for the account to be connected again).
type Retention struct {
	HTTPStatus int             `json:"httpStatus"`
	Body       json.RawMessage `json:"body"`
	ReadAt     time.Time       `json:"readAt"`
	AgeHours   float64         `json:"ageHours"`
}

// Whether it holds a curve with at least one point.
func (r *Retention) Curve() bool {
	if r == nil || r.HTTPStatus != 200 {
		return false
	}
	var said struct {
		RetentionCurve []json.RawMessage `json:"retentionCurve"`
	}
	return json.Unmarshal(r.Body, &said) == nil && len(said.RetentionCurve) > 0
}

// Why there is no curve, in a few words: "412 youtube_analytics_scope_missing", or "empty" for a video YouTube has not
// measured yet (it takes 2 to 3 days, and a few views).
func (r *Retention) Why() string {
	if r.HTTPStatus == 200 {
		return "empty"
	}
	var said struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(r.Body, &said)
	if said.Code == "" {
		return strconv.Itoa(r.HTTPStatus)
	}
	return strconv.Itoa(r.HTTPStatus) + " " + said.Code
}
