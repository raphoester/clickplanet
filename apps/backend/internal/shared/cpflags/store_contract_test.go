package cpflags_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
)

// What each module that keeps a catalog migrates in its own schema.
var migrations = fstest.MapFS{
	"1_heard_flags.up.sql": {Data: []byte(`
		CREATE TYPE heard_flag_state AS ENUM ('open', 'closed');
		CREATE TABLE heard_flags (
			flag       text             PRIMARY KEY,
			state      heard_flag_state NOT NULL,
			decided_at timestamptz      NOT NULL
		);
	`)},
	"1_heard_flags.down.sql": {Data: []byte(`DROP TABLE heard_flags; DROP TYPE heard_flag_state;`)},
}

func TestTheMemoryStoreKeepsTheContract(t *testing.T) {
	suite.Run(t, &cpflags.StoreContractSuite{
		NewStore: func() cpflags.Store { return cpflags.NewMemoryStore() },
	})
}

func TestThePostgresStoreKeepsTheContract(t *testing.T) {
	server := cppgtest.StartServer(t)
	suite.Run(t, &cpflags.StoreContractSuite{
		NewStore: func() cpflags.Store {
			db := server.CreateDatabase(t).OpenSchema(t, "flags_heard", migrations)
			require.NotNil(t, db)
			return cpflags.NewPostgresStore(db)
		},
	})
}
