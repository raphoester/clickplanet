package inmemory_tile_storage

type tileState struct {
	owner   uint16
	shields uint8
	holder  uint32
}

func heldFor(owner uint16, holder uint32) tileState {
	return tileState{owner: owner, holder: holder}
}
