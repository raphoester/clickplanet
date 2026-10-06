package inmemory_tile_storage

type tileState struct {
	owner   uint16
	shields uint8
}

func ownedBy(owner uint16) tileState {
	return tileState{owner: owner}
}
