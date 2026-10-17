package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const psqlSessions = 10

func TestTheProductionPoolsLeaveRoomForPsqlUnderMaxConnections(t *testing.T) {
	backend := loadYAML(t, "../../../../deploy/vps/backend.yaml")

	pools := 0
	for _, key := range backend.Keys() {
		block, ok := strings.CutSuffix(key, "database.host")
		if !ok {
			continue
		}
		maxOpen := block + "database.pool.maxOpenConns"
		require.True(t, backend.Exists(maxOpen), "%s is unbounded, so the sum below is no bound", maxOpen)
		pools += backend.Int(maxOpen)
	}

	assert.LessOrEqual(t, pools+psqlSessions, maxConnections(t, "../../../../deploy/vps/docker-compose.yaml"),
		"a psql session past max_connections is \"too many clients\" for every flush")
}

func maxConnections(t *testing.T, path string) int {
	t.Helper()

	for _, arg := range loadYAML(t, path).Strings("services.postgres.command") {
		if value, ok := strings.CutPrefix(arg, "max_connections="); ok {
			limit, err := strconv.Atoi(value)
			require.NoError(t, err)
			return limit
		}
	}
	require.FailNow(t, "postgres runs with the default max_connections", path)
	return 0
}

func loadYAML(t *testing.T, path string) *koanf.Koanf {
	t.Helper()

	k := koanf.New(".")
	require.NoError(t, k.Load(file.Provider(path), yaml.Parser()))
	return k
}
