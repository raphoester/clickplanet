package notes

type IDProvider interface {
	NewID() (NoteID, error)
}
