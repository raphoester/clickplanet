package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/parsers/yaml"
	kfile "github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconfigs"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

func TestTheExampleConfigStillReachesTheStructs(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	require.True(t, config.AntiBot.Enabled)

	assert.Equal(t, 5*time.Minute, config.Planet.Bonus.ForgetAfter)
	assert.Equal(t, time.Second, config.Planet.Bonus.SweepInterval)
	assert.Equal(t, time.Minute, config.Planet.RateLimiter.SweepInterval)

	assert.Equal(t, 72*time.Hour, config.Planet.Ledger.Retention)
	assert.Equal(t, 5*time.Minute, config.Planet.Ledger.SweepInterval)
	assert.Equal(t, time.Second, config.Planet.LedgerStorage.FlushInterval)

	assert.Equal(t, "antibot", config.AntiBot.Database.Schema)
	assert.Equal(t, 15*time.Minute, config.AntiBot.Jury.TrackWindow)

	assert.Equal(t, time.Minute, config.AntiBot.Evidence.SaveInterval)
	assert.Equal(t, 72*time.Hour, config.AntiBot.Evidence.Retention)

	assert.Equal(t, 5*time.Minute, config.AntiBot.Retaker.TrackWindow)
	assert.Equal(t, 10*time.Minute, config.AntiBot.Metronome.Stamina.Slice)
	assert.Equal(t, 15*time.Minute, config.AntiBot.Scraper.TrackWindow)
	assert.Equal(t, 44, config.AntiBot.Cohort.V6Bits)

	churner := config.AntiBot.Churner
	assert.Equal(t, 32, churner.Relay.V6Bits)
	assert.Equal(t, time.Hour+2*time.Minute, churner.TrackWindow)

	assert.Equal(t, 15*time.Minute, config.AntiBot.Hopper.TrackWindow)

	assert.Equal(t, 5*time.Minute, config.AntiBot.Rotator.Slot)
}

func TestTheExampleConfigStillCarriesTheRestOfTheFile(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "0.0.0.0:8080", config.HTTPServer.BindAddress)
	assert.Equal(t, time.Minute, config.HTTPServer.StartupTimeout)
	assert.NotZero(t, config.Planet.GameMap.MaxIndex)
	require.NoError(t, config.Planet.Validate())
	assert.Equal(t, time.Second, config.Planet.TilesStorage.FlushInterval)
	assert.Equal(t, "127.0.0.1:8081", config.HTTPServer.ConsoleBindAddress)
	assert.Equal(t, "127.0.0.1:8082", config.HTTPServer.InternalBindAddress)
	assert.Equal(t, "http://localhost:5173", config.HTTPServer.AllowedOrigin)
	assert.Equal(t, time.Hour, config.Auth.TTL)
}

func TestTheLocalStackConfigHasTheMapOfTheExampleAndAnInternalListener(t *testing.T) {
	var example, local Config
	require.NoError(t, cpconfigs.Load(&example, cpconfigs.FromFile("example.yaml")))
	require.NoError(t, cpconfigs.Load(&local, cpconfigs.FromFile("../../../../deploy/backend.yaml")))

	assert.Equal(t, example.Planet.GameMap.MaxIndex, local.Planet.GameMap.MaxIndex)
	assert.NotEmpty(t, local.HTTPServer.InternalBindAddress, "chat, player and scoring dial it while they are built")
}

// planet's rules are kept in postgres (planet.rule_sets): a rule left in a file would be read by nothing.
func TestEveryPlanetKeyIsOneTheConfigReads(t *testing.T) {
	theirs := reflect.TypeOf(Config{})
	for _, path := range []string{"example.yaml", "../../../../deploy/vps/backend.yaml", "../../../../deploy/backend.yaml"} {
		t.Run(path, func(t *testing.T) {
			file := koanf.New(".")
			require.NoError(t, file.Load(kfile.Provider(path), yaml.Parser()))
			for _, key := range file.MapKeys("") {
				for i := range theirs.NumField() {
					if field := theirs.Field(i); field.Name != "Planet" && strings.EqualFold(field.Name, key) {
						file.Delete(key)
					}
				}
			}

			var config planet.Config
			require.NoError(t, file.UnmarshalWithConf("", &config, koanf.UnmarshalConf{
				DecoderConfig: &mapstructure.DecoderConfig{
					DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
					ErrorUnused:      true,
					WeaklyTypedInput: true,
					Result:           &config,
				},
			}))
		})
	}
}

func authBlock(t *testing.T, secret string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, fmt.Appendf(nil, `
httpServer:
  bindAddress: 0.0.0.0:8080
  internalBindAddress: 127.0.0.1:8082
  allowedOrigin: https://clickplanet.lol
gameMap:
  maxIndex: 100
database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: planet}
redis: {address: localhost:6379}
chat:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: chat}
  redis: {address: localhost:6379}
player:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: player}
  redis: {address: localhost:6379}
scoring:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: scoring}
  redis: {address: localhost:6379}
gamemaster:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: gamemaster}
  redis: {address: localhost:6379}
missions:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: missions}
  redis: {address: localhost:6379}
support:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: support}
  redis: {address: localhost:6379}
clips:
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: clips}
  redis: {address: localhost:6379}
auth:
  enabled: true
  enforce: true
  secret: "%s"
  ttl: 2h
  database: {host: localhost, port: "5432", user: postgres, dbName: postgres, sslMode: disable, schema: auth}
  redis: {address: localhost:6379}
`, secret), 0o600))

	return path
}

func TestOnlyAuthReadsTheSeedOutOfTheAuthBlock(t *testing.T) {
	secret, _ := cpsession.TestKeyPair()

	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(authBlock(t, secret))))

	assert.Equal(t, secret, config.Auth.Secret)
	assert.Equal(t, 2*time.Hour, config.Auth.TTL)
	assert.True(t, config.Planet.Auth.Enabled)
	assert.True(t, config.Planet.Auth.Enforce)

	planetBlock := reflect.TypeOf(config.Planet.Auth)
	for i := range planetBlock.NumField() {
		assert.NotContains(t, strings.ToLower(planetBlock.Field(i).Name), "secret",
			"the planet block must carry nothing that could hold a signing key")
	}
}

func TestASecretThatIsNotASeedIsRefused(t *testing.T) {
	var config Config
	err := cpconfigs.Load(&config, cpconfigs.FromFile(authBlock(t, "not-a-seed")))
	require.ErrorContains(t, err, "auth.secret")
}

func TestTheExampleConfigReachesTheDatabaseBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "localhost", config.Planet.Database.Host)
	assert.Equal(t, "5432", config.Planet.Database.Port)
	assert.Equal(t, "postgres", config.Planet.Database.DBName)
	assert.Equal(t, "disable", config.Planet.Database.SSLMode)
	assert.Equal(t, "planet", config.Planet.Database.Schema)
	require.NotNil(t, config.Planet.Database.Pool.MaxOpenConns)
	assert.Equal(t, 4, *config.Planet.Database.Pool.MaxOpenConns)
}

func TestTheExampleConfigReachesThePlayerBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "player", config.Player.Database.Schema)
	require.NoError(t, config.Player.Database.Validate())
}

func TestTheExampleConfigReachesTheMissionsBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "missions", config.Missions.Database.Schema)
	require.NoError(t, config.Missions.Validate())
}

func TestTheMissionsModuleWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, Config{}.Validate(), "missions.database")
}

func TestThePlayerModuleWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, Config{}.Validate(), "player.database")
}

func TestTheProcessRefusesToStartWithoutADatabase(t *testing.T) {
	config := Config{}
	config.HTTPServer.BindAddress = "0.0.0.0:8080"
	config.Planet.GameMap.MaxIndex = 100

	require.ErrorContains(t, config.Validate(), "database: [host port user dbName sslMode schema] is empty")
}

func TestTheExampleConfigReachesTheChatDatabaseBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "chat", config.Chat.Database.Schema)
	assert.Equal(t, "localhost", config.Chat.Database.Host)
	assert.Equal(t, 720*time.Hour, config.Chat.Storage.Retention)
	require.NoError(t, config.Chat.Validate())
}

func TestChatWithoutADatabaseIsRefused(t *testing.T) {
	require.ErrorContains(t, Config{}.Validate(), "chat.database: [host port user dbName sslMode schema] is empty")
}

func TestAuthWithoutASecretIsRefused(t *testing.T) {
	config := Config{}
	config.HTTPServer.BindAddress = "0.0.0.0:8080"
	config.Planet.GameMap.MaxIndex = 100
	config.Auth.Enabled = true

	require.ErrorContains(t, config.Validate(), "auth.secret is empty")
}

func TestTheExampleConfigReachesTheAuthBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.False(t, config.Auth.Enabled, "the example must ship without auth")
	assert.Equal(t, "auth", config.Auth.Database.Schema)
	assert.Equal(t, 90*24*time.Hour, config.Auth.Sessions.GuestTTL)
	assert.Equal(t, 24*time.Hour, config.Auth.Sessions.ExtendEvery)
	assert.Equal(t, "session", config.Auth.Turnstile.Action)
	assert.Equal(t, 10, config.Auth.RateLimiter.Burst)
	assert.Equal(t, 30*24*time.Hour, config.Auth.Sessions.LinkedTTL)
	assert.False(t, config.Auth.SignIn.Enabled, "the example must ship without sign-in")
	assert.Equal(t, "http://localhost:5173/auth/callback", config.Auth.SignIn.RedirectURL)
	assert.Equal(t, 90*24*time.Hour, config.Auth.Prune.IdleFor)
	assert.Equal(t, time.Hour, config.Auth.Prune.Interval)
}

func TestTheProviderSecretsComeFromTheEnvironment(t *testing.T) {
	t.Setenv("auth.google.clientSecret", "google-secret")
	t.Setenv("auth.discord.clientSecret", "discord-secret")

	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "google-secret", config.Auth.Google.ClientSecret)
	assert.Equal(t, "discord-secret", config.Auth.Discord.ClientSecret)
}

func TestAuthWithoutADatabaseIsRefused(t *testing.T) {
	config := Config{}
	config.Auth.Enabled = true
	config.Auth.Secret = "a-secret"

	require.ErrorContains(t, config.Validate(), "auth.database: [host port user dbName sslMode schema] is empty")
}

func TestTheExampleConfigReachesTheScoringBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "scoring", config.Scoring.Database.Schema)
	require.NoError(t, config.Scoring.Database.Validate())
}

func TestTheScoringModuleWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, Config{}.Validate(), "scoring.database")
}

func TestTheExampleConfigReachesTheGamemasterBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "gamemaster", config.Gamemaster.Database.Schema)
	assert.Equal(t, 21*time.Hour, config.Gamemaster.DailyCutoff, "the days end at 21:00 UTC")
	require.NoError(t, config.Gamemaster.Validate())
}

func TestTheGamemasterModuleWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, Config{}.Validate(), "gamemaster.database")
}

func TestTheExampleConfigReachesTheSupportBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "support", config.Support.Database.Schema)
	assert.Equal(t, 90*24*time.Hour, config.Support.Retention)
	assert.Equal(t, time.Hour, config.Support.PruneInterval)
	assert.InDelta(t, 1.0/60, config.Support.RateLimiter.PerSecond, 1e-4, "one a minute")
	assert.Equal(t, 3, config.Support.RateLimiter.Burst)
	require.NoError(t, config.Support.Validate())
}

func TestTheSupportModuleWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, Config{}.Validate(), "support.database")
}

func TestTheExampleConfigReachesTheClipsBlock(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	assert.Equal(t, "clips", config.Clips.Database.Schema)
	assert.Equal(t, "localhost:6379", config.Clips.Redis.Address)
	require.NoError(t, config.Clips.Validate())
}

func TestTheClipsModuleWithNoDatabaseIsRefused(t *testing.T) {
	assert.ErrorContains(t, Config{}.Validate(), "clips.database")
}

func TestTheExampleConfigReachesTheAttachments(t *testing.T) {
	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile("example.yaml")))

	attachments := config.Support.Attachments
	assert.False(t, attachments.Enabled, "the example has no bucket")
	assert.Equal(t, 10, attachments.MaxPerNote)
	assert.Equal(t, int64(50<<20), attachments.MaxBytes)
	assert.Equal(t, 10*time.Minute, attachments.UploadTTL)
	assert.Equal(t, time.Hour, attachments.LinkTTL)
	assert.Equal(t, "auto", attachments.Bucket.Region)
	assert.Equal(t, "clickplanet-support", attachments.Bucket.Name)
}

func productionConfig(t *testing.T) Config {
	t.Helper()

	const production = "../../../../deploy/vps/backend.yaml"
	file, err := os.ReadFile(production)
	require.NoError(t, err)
	seed, _ := cpsession.TestKeyPair()
	for _, anchor := range regexp.MustCompile(`env://([A-Za-z_][A-Za-z0-9_]*)`).FindAllStringSubmatch(string(file), -1) {
		t.Setenv(anchor[1], "a-secret")
	}
	t.Setenv("SESSION_SECRET", seed)
	t.Setenv("SUPPORT_BUCKET_SECRET", "the-bucket-secret")
	t.Setenv("httpServer.allowedOrigin", "https://clickplanet.lol")

	var config Config
	require.NoError(t, cpconfigs.Load(&config, cpconfigs.FromFile(production)))
	return config
}

// The antibot's rules are kept in postgres (antibot.rule_sets): a rule left in a file would be read by nothing.
func TestEveryAntibotKeyIsOneTheConfigReads(t *testing.T) {
	for _, path := range []string{"example.yaml", "../../../../deploy/vps/backend.yaml", "../../../../deploy/backend.yaml"} {
		t.Run(path, func(t *testing.T) {
			file := koanf.New(".")
			require.NoError(t, file.Load(kfile.Provider(path), yaml.Parser()))

			var config antibot.Config
			require.NoError(t, file.UnmarshalWithConf("antiBot", &config, koanf.UnmarshalConf{
				DecoderConfig: &mapstructure.DecoderConfig{
					DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
					ErrorUnused:      true,
					WeaklyTypedInput: true,
					Result:           &config,
				},
			}))
		})
	}
}

func TestTheProductionConfigTakesFilesInItsBucketAndNeedsItsSecret(t *testing.T) {
	config := productionConfig(t)

	attachments := config.Support.Attachments
	assert.True(t, attachments.Enabled)
	assert.Equal(t, "the-bucket-secret", attachments.Bucket.SecretAccessKey)
	assert.Equal(t, "9ec4cd4fdf48006b2944c522219f1eea", attachments.Bucket.AccessKeyID)
	assert.Equal(t, "https://76f6f3883c3a531ae90d1dc4b96bf810.r2.cloudflarestorage.com", attachments.Bucket.Endpoint)
	assert.Equal(t, "clickplanet-support", attachments.Bucket.Name)
	assert.Equal(t, "auto", attachments.Bucket.Region)
	assert.Equal(t, int64(9<<30), attachments.MaxStoredBytes)
	require.NoError(t, config.Support.Validate())

	config.Support.Attachments.Bucket.SecretAccessKey = ""
	require.ErrorContains(t, config.Support.Validate(), "secretAccessKey",
		"a deploy before SUPPORT_BUCKET_SECRET is set refuses the boot")
}

func TestTheProductionVideosShareTheSupportBucketAndItsKey(t *testing.T) {
	config := productionConfig(t)

	videos := config.Clips.Videos
	assert.True(t, videos.Enabled)
	assert.Equal(t, config.Support.Attachments.Bucket, videos.Bucket)
	require.NoError(t, config.Clips.Validate())

	config.Clips.Videos.Bucket.SecretAccessKey = ""
	require.ErrorContains(t, config.Clips.Validate(), "clips.videos.bucket")
}

func TestEveryModuleOfProductionTalksToTheSameRedis(t *testing.T) {
	config := productionConfig(t)

	for name, redis := range map[string]cpredis.Config{
		"auth": config.Auth.Redis, "chat": config.Chat.Redis, "player": config.Player.Redis,
		"scoring": config.Scoring.Redis, "support": config.Support.Redis, "missions": config.Missions.Redis,
		"antiBot": config.AntiBot.Redis, "gamemaster": config.Gamemaster.Redis, "clips": config.Clips.Redis,
	} {
		assert.Equal(t, config.Planet.Redis, redis, "%s.redis is the planet's block", name)
	}
	assert.Equal(t, "redis:6379", config.Planet.Redis.Address)
	assert.Equal(t, "a-secret", config.Planet.Redis.Password, "from REDIS_PASSWORD")
}

func TestTheAntibotKeepsItsKeysOutsideThePlanetsBlock(t *testing.T) {
	config := productionConfig(t)

	require.True(t, config.AntiBot.Enabled)
	assert.Equal(t, "antibot", config.AntiBot.Database.Schema)
	assert.Equal(t, 10*time.Minute, config.AntiBot.Defender.TrackWindow)
	require.NoError(t, config.AntiBot.Validate())
}
