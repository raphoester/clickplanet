package messages

type Record struct {
	message   Message
	authorID  string
	ip        string
	userAgent string
}

type RecordSnapshot struct {
	Message   Snapshot
	AuthorID  string
	IP        string
	UserAgent string
}

const (
	maxAuthorIDLength  = 64
	maxUserAgentLength = 256
)

func NewRecord(message Message, authorID string, ip string, userAgent string) Record {
	return Record{
		message:   message,
		authorID:  truncate(authorID, maxAuthorIDLength),
		ip:        ip,
		userAgent: truncate(userAgent, maxUserAgentLength),
	}
}

func (r Record) TakeSnapshot() RecordSnapshot {
	return RecordSnapshot{
		Message:   r.message.TakeSnapshot(),
		AuthorID:  r.authorID,
		IP:        r.ip,
		UserAgent: r.userAgent,
	}
}

func truncate(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes]
}
