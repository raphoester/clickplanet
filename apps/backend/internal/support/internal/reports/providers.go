package reports

type IDProvider interface {
	NewID() (ReportID, error)
}
