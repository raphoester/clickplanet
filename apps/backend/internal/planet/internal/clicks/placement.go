package clicks

import "github.com/google/uuid"

// PlacementID names a shield on its way from a hand to a tile: taken from the hand, and not yet saved on the map.
type PlacementID uuid.UUID

func (id PlacementID) String() string { return uuid.UUID(id).String() }
