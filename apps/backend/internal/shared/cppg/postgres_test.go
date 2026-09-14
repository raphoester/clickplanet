package cppg_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
)

func TestThePasswordStaysOutOfAPrintedConfig(t *testing.T) {
	config := struct{ Database cppg.Config }{cppg.Config{Host: "postgres", Password: "hunter2"}}

	printed := fmt.Sprintf("%+v", config)

	assert.NotContains(t, printed, "hunter2")
	assert.Contains(t, printed, "postgres")
}

func TestAnIncompleteConfigNamesEveryMissingKey(t *testing.T) {
	err := cppg.Config{Host: "postgres", Port: "5432"}.Validate()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database.user database.dbName database.sslMode")
}

func TestACompleteConfigIsValid(t *testing.T) {
	require.NoError(t, cppg.Config{Host: "h", Port: "5432", User: "u", DBName: "d", SSLMode: "disable"}.Validate())
}
