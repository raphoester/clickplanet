package notes

type IDProvider interface {
	NewID() (NoteID, error)
}

type AttachmentIDProvider interface {
	NewAttachmentID() (AttachmentID, error)
}
