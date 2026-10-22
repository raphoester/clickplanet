package tools_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/raphoester/clickplanet.lol-backend/generated/descriptors"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools"
)

var (
	leftOut = []protoreflect.FullName{"auth.v1"}
	hidden  = []protoreflect.FullName{"planet.v1.AdminService.GetReplay"}
)

func catalog(t *testing.T) *tools.Catalog {
	t.Helper()
	catalog, err := tools.CatalogOf(descriptors.Admin(), leftOut, hidden)
	require.NoError(t, err)
	return catalog
}

func TestEveryAdminMethodIsATool(t *testing.T) {
	names := map[string]string{}
	for _, tool := range catalog(t).Tools() {
		names[tool.Name] = tool.Procedure
	}

	assert.Equal(t, "/planet.v1.AdminService/BanPlayer", names["planet_ban_player"])
	assert.Equal(t, "/chat.v1.AdminService/RemoveMessage", names["chat_remove_message"])
	assert.Equal(t, "/player.v1.AdminService/ReconcileTitles", names["player_reconcile_titles"])
	assert.Equal(t, "/support.v1.AdminService/DecideReport", names["support_decide_report"])
	assert.NotContains(t, names, "planet_get_replay", "hidden: megabytes for a day")
	for name := range names {
		assert.NotRegexp(t, "^auth_", name, "an app never signs in or lets an app in")
	}
}

// The tools are read from the embedded set and served by the generated code: both come from one `make proto`.
func TestTheEmbeddedSetIsTheOneTheServerIsBuiltFrom(t *testing.T) {
	served := 0
	for _, tool := range catalog(t).Tools() {
		name := strings.ReplaceAll(strings.TrimPrefix(tool.Procedure, "/"), "/", ".")
		_, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name))
		require.NoError(t, err, "%s is in the set and not in the generated code: run make proto", tool.Procedure)
		served++
	}

	generated := 0
	for _, pkg := range []string{"planet.v1", "chat.v1", "player.v1", "support.v1"} {
		service, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(pkg + ".AdminService"))
		require.NoError(t, err)
		generated += service.(protoreflect.ServiceDescriptor).Methods().Len()
	}
	assert.Equal(t, generated-len(hidden), served, "a method in the generated code and not in the set: run make proto")
}

func TestEveryToolSaysWhatItDoes(t *testing.T) {
	for _, tool := range catalog(t).Tools() {
		assert.NotEmpty(t, tool.Description, "%s has no comment in its admin.proto: it is the tool's description", tool.Procedure)
	}
}

func TestAReadIsAMethodMarkedNoSideEffects(t *testing.T) {
	c := catalog(t)

	for name, reads := range map[string]bool{
		"planet_top_players": true, "planet_inspect_player": true, "support_list_notes": true, "chat_get_message": true,
		"planet_ban_player": false, "chat_mute": false, "planet_reassign_country": false,
	} {
		tool, err := c.Tool(name)
		require.NoError(t, err, name)
		assert.Equal(t, reads, tool.ReadOnly, name)
	}

	_, err := c.Tool("planet_drop_database")
	require.ErrorIs(t, err, tools.ErrUnknownTool)
}

func TestAToolsInputIsItsRequestAsProtojsonReadsIt(t *testing.T) {
	tool, err := catalog(t).Tool("planet_ban_player")
	require.NoError(t, err)
	assert.Equal(t, "Ban player", tool.Title)

	encoded, err := json.Marshal(tool.InputSchema)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"type": "object",
		"additionalProperties": false,
		"properties": {
			"scope": {"type": "string", "description": "An address, banned as its scope. Never with account_id."},
			"accountId": {"type": "string", "description": "Never with scope."},
			"duration": {"type": "string", "pattern": "^-?[0-9]+(\\.[0-9]+)?s$", "examples": ["3600s"],
				"description": "Unset is the ladder's step for this offence."}
		}
	}`, string(encoded))

	notes, err := catalog(t).Tool("support_list_notes")
	require.NoError(t, err)
	properties := notes.InputSchema["properties"].(map[string]any)
	assert.Equal(t, tools.Schema{"type": "string", "format": "date-time", "description": "Unset is every note kept."}, properties["since"])
	assert.Equal(t, []string{"NOTE_KIND_UNSPECIFIED", "NOTE_KIND_SUGGESTION", "NOTE_KIND_BUG"}, properties["kind"].(tools.Schema)["enum"])
	assert.Equal(t, 0, properties["limit"].(tools.Schema)["minimum"])
}
