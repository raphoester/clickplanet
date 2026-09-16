package messages

// Event is one thing that happened in the chat, as the storage fans it out:
// exactly one field is set. An envelope for the reason ChatEvent is one on the
// wire — a second kind of live event must not need a second subscription, and a
// redaction sent down another channel could overtake the message it blanks.
type Event struct {
	Message   *Message
	Redaction *Redaction
}

// Redaction blanks everything one author said, on the screens already showing it.
type Redaction struct {
	AuthorTag string
}
