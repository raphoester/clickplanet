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
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/alerts/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/promo/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/publishing/v1"
	_ "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/mcp/internal/tools"
)

func catalog(t *testing.T) *tools.Catalog {
	t.Helper()
	catalog, err := tools.CatalogOf(descriptors.Admin(), tools.LeftOut, tools.Hidden)
	require.NoError(t, err)
	return catalog
}

func TestEveryAdminMethodIsATool(t *testing.T) {
	names := map[string]string{}
	for _, tool := range catalog(t).Tools() {
		names[tool.Name] = tool.Procedure
	}

	assert.Equal(t, "/antibot.v1.AdminService/BanPlayer", names["antibot_ban_player"])
	assert.Equal(t, "/chat.v1.AdminService/RemoveMessage", names["chat_remove_message"])
	assert.Equal(t, "/missions.v1.AdminService/ListFlagDays", names["missions_list_flag_days"])
	assert.Equal(t, "/missions.v1.AdminService/SaveRules", names["missions_save_rules"])
	assert.Equal(t, "/progression.v1.AdminService/ReconcileTitles", names["progression_reconcile_titles"])
	assert.Equal(t, "/support.v1.AdminService/DecideReport", names["support_decide_report"])
	for _, hidden := range []string{
		"planet_get_replay", "planet_get_ledger", "planet_get_territories",
		"clips_keep_snapshot", "clips_make_cast", "clips_keep_appearance",
	} {
		assert.NotContains(t, names, hidden)
	}
	for name := range names {
		assert.NotRegexp(t, "^auth_", name, "an app never signs in or lets an app in")
	}
}

func TestTheEmbeddedSetIsTheOneTheServerIsBuiltFrom(t *testing.T) {
	served := 0
	for _, tool := range catalog(t).Tools() {
		name := strings.ReplaceAll(strings.TrimPrefix(tool.Procedure, "/"), "/", ".")
		_, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name))
		require.NoError(t, err, "%s is in the set and not in the generated code: run make proto", tool.Procedure)
		served++
	}

	offered := 0
	for _, pkg := range []string{
		"alerts.v1", "antibot.v1", "chat.v1", "clips.v1", "gamemaster.v1", "missions.v1", "planet.v1", "player.v1",
		"progression.v1", "promo.v1", "publishing.v1", "support.v1",
	} {
		service, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(pkg + ".AdminService"))
		require.NoError(t, err)
		offered += service.(protoreflect.ServiceDescriptor).Methods().Len()
	}
	assert.Equal(t, offered-len(tools.Hidden), served, "a method in the generated code and not in the set: run make proto")
}

func TestEveryToolSaysWhatItDoes(t *testing.T) {
	for _, tool := range catalog(t).Tools() {
		assert.NotEmpty(t, tool.Description, "%s has no comment in its admin.proto: it is the tool's description", tool.Procedure)
	}
}

func TestAReadIsAMethodMarkedNoSideEffects(t *testing.T) {
	c := catalog(t)

	for name, reads := range map[string]bool{
		"planet_top_players": true, "antibot_inspect_player": true, "support_list_notes": true, "chat_get_message": true,
		"antibot_ban_player": false, "chat_mute": false, "planet_reassign_country": false,
		"chat_get_raid_mode": true, "chat_set_raid_mode": false,
	} {
		tool, err := c.Tool(name)
		require.NoError(t, err, name)
		assert.Equal(t, reads, tool.ReadOnly, name)
	}

	_, err := c.Tool("planet_drop_database")
	require.ErrorIs(t, err, tools.ErrUnknownTool)
}

func TestAToolsInputIsItsRequestAsProtojsonReadsIt(t *testing.T) {
	tool, err := catalog(t).Tool("antibot_ban_player")
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
