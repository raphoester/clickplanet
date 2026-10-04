package messages

type Record struct {
	message   Message
	authorID  string
	ip        string
	userAgent string
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

func (r Record) Message() Message { return r.message }

func (r Record) AuthorID() string { return r.authorID }

func (r Record) IP() string { return r.ip }

func (r Record) UserAgent() string { return r.userAgent }

func truncate(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes]
}
