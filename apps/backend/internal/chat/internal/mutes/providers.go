package mutes

type IDProvider interface {
	NewID() (MuteID, error)
}
