//go:build testing

package clicks

// TestBoard is earth-1's row, as the migration that made boards keeps it: what a test names the board by.
const TestBoard BoardID = 1

// WorldStage is the world's map on TestBoard.
var WorldStage = World.On(TestBoard)
