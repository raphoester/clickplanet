package send_message_usecase

type Config struct {
	MaxTextLength int
	// Refused at send, added to messages.DefaultBannedPhrases.
	BannedPhrases []string
}
