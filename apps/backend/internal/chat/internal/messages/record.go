package messages

import "strings"

type Record struct {
	message   Message
	ip        string
	userAgent string
}

type RecordSnapshot struct {
	Message   Snapshot
	IP        string
	UserAgent string
}

const maxUserAgentLength = 256

func NewRecord(message Message, ip string, userAgent string) Record {
	return Record{
		message:   message,
		ip:        ip,
		userAgent: truncate(userAgent, maxUserAgentLength),
	}
}

func (r Record) TakeSnapshot() RecordSnapshot {
	return RecordSnapshot{
		Message:   r.message.TakeSnapshot(),
		IP:        r.ip,
		UserAgent: r.userAgent,
	}
}

func truncate(value string, maxBytes int) string {
	value = strings.ToValidUTF8(value, "")
	if len(value) <= maxBytes {
		return value
	}
	return strings.ToValidUTF8(value[:maxBytes], "")
}
