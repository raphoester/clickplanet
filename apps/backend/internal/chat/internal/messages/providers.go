package messages

type IDProvider interface {
	NewID() (MessageID, error)
}
