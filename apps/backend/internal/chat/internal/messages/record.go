package messages

// Record is a message plus what the log keeps about its sender and no reader sees.
type Record struct {
	Message   Message
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
		Message:   message,
		AuthorID:  truncate(authorID, maxAuthorIDLength),
		IP:        ip,
		UserAgent: truncate(userAgent, maxUserAgentLength),
	}
}

func truncate(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes]
}
