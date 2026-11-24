package catalog_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
)

func TestANameAnotherFlagHasIsTakenWhateverItsCase(t *testing.T) {
	names := catalog.NamesOf([]catalog.Flag{catalog.OpenFlag("r-0000000001", "Bavaria", openedAt)})

	require.ErrorIs(t, names.Free("brittany", ""), catalog.ErrNameTaken, "a flag the game ships with")
	require.ErrorIs(t, names.Free("FRANCE", "r-0000000001"), catalog.ErrNameTaken)
	require.ErrorIs(t, names.Free("BAVARIA", ""), catalog.ErrNameTaken, "a flag opened since")
	require.ErrorIs(t, names.Free("bavaria", "r-0000000002"), catalog.ErrNameTaken)
	require.NoError(t, names.Free("BAVARIA", "r-0000000001"), "its own name, in another case")
	require.NoError(t, names.Free("Savoy", ""))
}
