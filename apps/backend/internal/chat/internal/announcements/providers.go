package announcements

type IDProvider interface {
	NewID() (AnnouncementID, error)
}
