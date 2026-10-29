# Configuration

**`shared/cpconfigs` owns loading**, the way `cpbootstrap` owns running: the binary asks for the config it wants and never for the flag, the parser, or the precedence between file and environment.

```go
var config Config
err := configs.Load(&config, configs.FromFlag())
```

Where the file comes from is an option — `FromFlag()` reads `-config`, which is how the container runs it; `FromFile(path)` names one outright and lives behind the `testing` tag, because two test packages need it and no production caller does (see [Testing](testing.md)). **An empty path is not an error**: every field keeps its zero value and the environment alone can carry a whole config.

Config is loaded from a YAML file, with environment variables overriding it — `.` is the nesting delimiter, so `database.password=...` in the environment overrides the file. See `cmd/api/example.yaml` for the full schema.

**A string value may be an anchor rather than a literal.** `secret: env://SESSION_SECRET` reads that variable at load; anything carrying no known scheme is left exactly as written. This is what keeps a config file that is in git self-contained — it names where each secret comes from, at the point the secret is used, instead of a table elsewhere mapping one name onto another. `EnvResolver` is the only scheme today; a second one is another `SecretResolver` in `cpconfigs` and no change to any config struct or call site.

**An anchored variable that is not set fails the load**, naming the field and the variable. That is deliberately not the same as one set to the empty string, which is a value and reaches the block's own `Validate` — so `SESSION_SECRET` missing is a fault, and `SESSION_SECRET=` is a setting the session block then refuses on its own terms.

**A config that implements `Validate() error` is asked to check itself**, and the load fails with its sentence wrapped in `cpconfigs.ErrValidation`. That is where a bad setting is refused out loud rather than becoming a zero value nothing reports.

**Every block validates its own, and `cmd/api` only fans out:**

```go
func (c Config) Validate() error {
	return errors.Join(
		c.HTTPServer.Validate(),
		c.Clicks.Validate(),
		c.Auth.Validate(),
		c.Chat.Validate(),
		c.Player.Validate(),
	)
}
```

The binary never reads inside a block to check it, so a new bound is added in the module that owns it and nothing here changes. `errors.Join` also means a broken file reports **everything** wrong at once rather than one line per restart.

- `cpbootstrap.ServerConfig` — `bindAddress` empty listens on port 80; `consoleBindAddress` set to anything but loopback
- `shared/cpredis.Config` — an `address` that is not a `host:port`, or a negative limit. Each module checks its own `redis` block, as it checks its database block
- `shared/cppg.Config` — a connection setting or the schema left empty, or a schema that is not a plain lowercase identifier. Each module checks its own block, starting with `planet.Config`
- `planet.Config` — `gameMap.maxIndex` zero is a map that refuses every click, plus whatever `bonus` refuses of its own
- `antibot.Config` — with `antiBot.enabled`, an incomplete `antiBot.database` or `antiBot.redis` block, and the cohort's prefix bits. Its rules are checked when a set is saved and when one is read (see [The antibot's rules](antibot-rules.md))
- `shared/cpsession.SignerConfig` — `secret` empty, not hex or not 32 bytes while `enabled`, and a negative `ttl`. Checked by `auth.Config`, which is the only block that declares it. `cpsession.VerifierConfig`, which planet declares, has nothing to check: two switches and no key
- `chat.Config` — an incomplete `chat.database` block. Every other chat setting has a usable default
- `support.Config` — an incomplete `support.database` block; with `support.attachments.enabled`, an incomplete bucket block, or a link good for more than 7 days. Every other support setting has a usable default

There is no struct-tag validation and therefore no validator dependency — a hook the config implements covers this app's needs.

**Each module owns its own config struct** — `planet.Config`, `chat.Config`, `auth.Config`, `player.Config` — and `app.Config` is the four of them plus `httpServer`. The planet keys stayed at the top level of the file rather than moving under a `planet:` section: `app.Config` squashes that struct (`koanf:",squash"`), so the file and every `deploy/` environment variable are unchanged.

- `httpServer.bindAddress` — the encoding is negotiated per request, so there is no format setting.
- `redis` — each module's own block (`redis` at the top level for planet, `auth.redis`, `antiBot.redis`, `chat.redis`, `player.redis`, `scoring.redis`, `support.redis`, `missions.redis`, `gamemaster.redis`, `clips.redis`): the Redis it tells the others what happened through and hears them on (see [Telling other modules what happened](composite-layer.md#telling-other-modules-what-happened)). **Every block names the same Redis**, or a module publishes where nobody listens; `TestEveryModuleOfProductionTalksToTheSameRedis` pins it for `deploy/vps/backend.yaml`. Each module connects on its own, and one that cannot reach Redis refuses the boot
- `redis.address`, `redis.password` — production reads the password from `REDIS_PASSWORD`
- `redis.maxLen` — the most entries one stream keeps (100k), for a stream whose subscriber stopped; the trim keeps them far fewer while every subscriber keeps up
- `redis.publishTimeout` — how long a click or any other publish waits on Redis (500ms); past it the event is lost and counted
- `redis.retryAfter`, `redis.maxDeliveries` — the first wait before a module's failed event is handled again (2s, doubling up to 64 times that), and how many tries before it is buried in `events:dead` (8)
- `httpServer.streamHeartbeat` — how often a silent live stream sends a heartbeat (default 30s), after the one it opens with. **Must stay well under the proxy's idle cut**: Cloudflare answers 524 at ~125s, and a stream that never speaks is one it kills.
- `httpServer.startupTimeout` — how long building every module may take, the loads included (default 1m). A boot that passes it fails, and the container's restart policy boots it again. See [Durability](durability.md) for why it is not 5s.
- `httpServer.consoleBindAddress` — where the operator services listen for a shell on the box, with no session (see [Operator tools](admin.md)); empty serves none, and a non-loopback address refuses the boot
- `httpServer.adminBindAddress` — where the admin services listen again, behind the admin gate, for the admin page (see [Operator tools](admin.md)); empty serves none, and set with no gate refuses the boot
- `auth.admin.origin` — the admin page's exact origin, the only one the gate answers; empty builds no gate and offers no admin sign-in, and set with `auth.enabled` false refuses the boot. In production it comes from `ADMIN_ORIGIN`
- `auth.admin.redirectUrl`, `auth.admin.google.clientId`, `auth.admin.google.clientSecret` — the admin page's own Google client and callback; with an origin, a missing one refuses the boot. The secret is `env://ADMIN_GOOGLE_CLIENT_SECRET`
- `auth.admin.sessionTTL` — how long an admin session lasts, never extended (12h)
- `httpServer.internalBindAddress` — where the services other modules call listen (see [Calling another module](composite-layer.md#calling-another-module)); empty serves none, a caller that dials it then fails, and a non-loopback address refuses the boot
- `httpServer.allowedOrigin` — the frontend's exact origin (`scheme://host[:port]`, no path). The public router answers CORS for it alone, with `Access-Control-Allow-Credentials: true`, so the `cp_sid` cookie travels on the web client's mint. **Empty, `*` or anything that is not an origin refuses the boot**: a browser drops a credentialed answer that allows every origin, so a wrong value would only show up as every mint starting a new guest. In production it comes from `FRONTEND_ORIGIN`, the value the Caddyfile allows, which sets the same headers and overwrites these.
- `gameMap.maxIndex` — total number of tiles
- `database.host`, `port`, `user`, `password`, `dbName`, `sslMode`, `schema`, `pool.*` — the planet module's postgres and the schema its tables live in; any of them but `password` and `pool` empty refuses the boot. `database.password` belongs in the environment
- `tilesStorage.flushInterval` — how often the tiles changed since the last flush are written to postgres (1s)
- `ledger.retention`, `ledger.sweepInterval` — how long the operator tools can trace and revert a take (72h)
- `ledgerStorage.flushInterval` — how often new takes are written to postgres (1s, and on shutdown)
- `ledgerStorage.maxTakes` — the most takes kept (4M, ~100 MiB); past it the oldest go before the retention
- `tilesStorage.subscriberBuffer` — per-subscriber channel capacity, which is now per connected client rather than per fanout; a subscriber that falls that far behind is closed, never blocked on or skipped
- **Every game rule is the rule set's, kept in postgres** (`planet.rule_sets`), not a key here: the refill, the bank
  and the scope multipliers, the toll, the boxes, the charges, the welcome and the quizzes. See [Rules](rules.md)
- `rateLimiter.sweepInterval` — how often the click limiter forgets a bucket that is full (1m)
- `vpnBlocklist.enabled`, `vpnBlocklist.includeDatacenters`, `vpnBlocklist.allow` — the VPN refusal (see [VPN blocklist](vpn-blocklist.md)); disabled parses nothing and allocates nothing
- `bonus.forgetAfter`, `bonus.sweepInterval` — how long a caller with no stream open is remembered (5m), and how often
  the boxes and the quizzes are swept (1s)
- `antiBot` — the antibot module's block (`antibot.Config`, beside `planet.Config` in `cmd/api`). It holds the machine only: **every bound the antibot judges with is a rule set in postgres**, edited from the backoffice (see [The antibot's rules](antibot-rules.md)). A rule left in the file is read by nothing, and `TestEveryAntibotKeyIsOneTheConfigReads` refuses it
- `antiBot.enabled` — off builds no module: nothing is heard, nothing is refused, and `antibot.v1.AdminService` 404s
- `antiBot.database` — the antibot's own `cppg.Config`, `schema: antibot`; required when `antiBot.enabled`. A failed connection, migration, rule set or load refuses the boot
- `antiBot.redis` — the antibot's own `cpredis.Config`, the same Redis as every module's: what it hears planet's facts on and sends planet's commands through
- `antiBot.jury.trackWindow`, `sweepInterval` — how long a silent caller is remembered, and how often the jury forgets and sets its gauges
- `antiBot.evidence.saveInterval`, `retention` — how often every watchdog's evidence and the jury's record are written to `antibot.evidence` (1m, and on shutdown), and the oldest kept (72h) on load and in memory. Also how long `UnbanPlayer` holds the jury off. See [What survives a restart](antibot.md#what-survives-a-restart)
- `antiBot.<watchdog>` — one block per watchdog (`retaker`, `sequencer`, `metronome`, `defender`, `catcher`, `cohort`, `scraper`, `churner`, `hopper`, `rotator`). Every watchdog is built: whether its reading counts is the rule `<watchdog>.enabled`
- every watchdog takes `trackWindow` and `sweepInterval` — how far back its evidence counts, and how often what can no longer matter is forgotten. The retaker, the sequencer, the defender, the scraper and the hopper count their rule over `trackWindow`; the cohort keeps at least the rules' `chainWindow`, the churner `window` plus `relay.handoff`. The catcher's must hold `minCatches` boxes at the slowest box pace plus the planet rule set's `bonuses.offerTtl`
- `antiBot.metronome.stamina.slice`, `antiBot.rotator.slot` — the length the evidence is counted in, saved with it: a section saved under another length is dropped on load
- `antiBot.cohort.v4Bits`, `v6Bits`, `antiBot.churner.relay.v4Bits`, `v6Bits` — the wider prefix a scope or an account is indexed by when first seen (a /24 and a /44, a /24 and a /32). A prefix wider than an address refuses the boot (the cohort's)
- `auth.enabled` — off registers nothing, so `auth.v1` and `session.v1` 404 and clicks are judged on address alone
- `auth.enforce` — off counts what enforcing would refuse without refusing it; the mode to deploy in
- `auth.secret` — the Ed25519 seed the tokens are signed with, 32 bytes as 64 hex characters (`openssl rand -hex 32`); **required once `auth.enabled` is true**, and an empty or malformed one refuses the boot rather than being invented. It is the only key in the file: `planet` asks `auth` for the verifying half over the internal listener
- `auth.ttl` — how long a minted token is accepted (default 1h)
- `auth.rateLimiter.*` — the per-IP throttle on both `CreateSession` paths together, same shape as `rateLimiter`
- `auth.resumeLimiter.*` — the per-IP throttle on `ResumeSession`, same shape; its defaults suit it
- `auth.turnstile.enabled` — off mints for anyone who asks, which is how a local backend runs without a widget
- `auth.turnstile.secret` — the widget's secret half, from the environment
- `auth.turnstile.hostnames` — the frontend origins siteverify must report; **empty refuses every token** rather than accepting any, and a production value must not include `localhost`
- `auth.turnstile.action` — must match the widget's `data-action` (default `session`)
- `auth.database.*` — accounts and their sessions, same shape as `database`, schema `auth`; required when `auth.enabled`. `auth.database.password` belongs in the environment
- `auth.sessions.guestTTL`, `auth.sessions.linkedTTL`, `auth.sessions.extendEvery` — how long an idle guest keeps its cookie (90 days), how long an idle signed-in account does (30 days), and how often using a session extends it (24h)
- `auth.signIn.enabled` — off, `StartSignIn` and `CompleteSignIn` answer `Unimplemented` (404). On, at least one provider needs a `clientId`
- `auth.signIn.redirectUrl` — the frontend callback page, registered with every provider exactly (production `https://clickplanet.lol/auth/callback`); not an absolute URL, or one with a query, refuses the boot
- `auth.google.clientId`, `auth.discord.clientId` — a provider is offered once its id is set
- `auth.google.clientSecret`, `auth.discord.clientSecret` — from the environment; empty beside a set `clientId` refuses the boot
- `auth.email.enabled` — off, `StartEmailSignIn` and `CompleteEmailSignIn` answer `Unimplemented` (404) and `GetSignInOptions` does not list email. Apart from `auth.signIn`: it needs no provider
- `auth.email.delivery` — `cloudflare`, or `log` for a local backend; anything else refuses the boot while email is on
- `auth.email.from`, `auth.email.fromName` — the sender, on a domain onboarded to Cloudflare Email Sending; not an address refuses the boot
- `auth.email.cloudflare.accountId` — the account the domain is onboarded on, written in `deploy/vps/backend.yaml` as it is: it is not a secret
- `auth.email.cloudflare.apiToken` — `env://CLOUDFLARE_EMAIL_TOKEN` in `deploy/vps/backend.yaml`; it or the account id empty, with email on and `delivery: cloudflare`, refuses the boot
- `auth.email.sendLimiter.*` — the per-address throttle on codes, same shape as `rateLimiter` (default 3, then one every 20 minutes)
- `auth.prune.idleFor`, `auth.prune.interval` — how long a guest goes unused before it is deleted (default `guestTTL`, never less), and how often the prune runs (1h)
- `chat.database.host`, `port`, `user`, `password`, `dbName`, `sslMode`, `schema`, `pool.*` — the chat module's postgres, same shape as `database`; any of them but `password` and `pool` empty refuses the boot. `chat.database.password` belongs in the environment
- `chat.storage.historySize`, `chat.storage.retention`, `chat.storage.pruneInterval`, `chat.storage.subscriberBuffer`
- `chat.service.maxTextLength` — the bound on a message in runes (280)
- `chat.rateLimiter.*` — the per-IP `SendMessage` throttle, same shape as `rateLimiter`
- `chat.reactionLimiter.*` — the per-IP `React` throttle, same shape; its defaults suit it
- `chat.seenLimiter.*` — the per-IP `MarkSeen` throttle, same shape; its defaults suit it
- `chat.blockedIPs` — prefixes refused every chat RPC, parsed by `shared/cpipblock` exactly as `vpnBlocklist.allow` is
- `player.tagSalt` — salts the hash of an address the roster caps its visits with. It is never shown, so empty, which generates one at boot, costs nothing. Production reads it from `CHAT_TAG_SALT`, the chat's old variable
- `player.database.*` — profiles and stats, same shape as `database`, schema `player`; required. `player.database.password` belongs in the environment
- `missions.database.*` — the briefs, the sheets and the collections, same shape as `database`, schema `missions`; required. `missions.database.password` belongs in the environment
- `scoring.database.*` — the standings and the rounds, same shape as `database`, schema `scoring`; required. `scoring.database.password` belongs in the environment
- `gamemaster.dailyCutoff` — the time of day, UTC, every day of a season ends at, as a duration from midnight (`21h`); 0s
  to under 24h. A day is a round the gamemaster tells scoring to score, and tells missions as it starts (see [The gamemaster module](gamemaster.md))
- `gamemaster.database.*` — the rule schedule's entries, the seasons, same shape as `database`, schema `gamemaster`; required. `gamemaster.database.password` belongs in the environment. See [The gamemaster module](gamemaster.md)
- `support.database.*` — what players send the operator, same shape as `database`, schema `support`; required. `support.database.password` belongs in the environment
- `clips.database.*` — the daily snapshots of the clips' numbers, the casts and who each post names, same shape as `database`, schema `clips`; required. `clips.database.password` belongs in the environment
- `support.retention`, `support.pruneInterval` — how long a note keeps its sender (default 90 days; the text is kept after) and how long a report is kept, and how often the prune runs (1h)
- `support.rateLimiter.*` — the per-IP throttle on the notes and the reports, same shape as `rateLimiter` (default one a minute, three in hand)
- `support.noteLimiter.*` — suggestions and bug reports a day, per account and per address together, same shape (default a burst of 10, refilled over a day: unset, `perSecond` is the burst over a day)
- `support.uploadLimiter.*` — files a day, per account and per address together, same shape (default 100 a day)
- `support.attachments.enabled` — off, a note carries no file and the bucket block is not read
- `support.attachments.maxPerNote`, `support.attachments.maxBytes` — the most files a note carries (10) and the most bytes of one (52428800, 50 MiB). The page reads both from `GetAttachmentRules`
- `support.attachments.maxStoredBytes` — the most bytes the bucket holds, the files notes carry and the files that may still come (9 GiB)
- `support.attachments.keepUnsent` — how long a file waits for the note that names it before it is deleted (24h); not past `uploadTTL` refuses the boot
- `support.attachments.uploadTTL`, `support.attachments.linkTTL` — how long an upload link is good for (10m), and a link the operator opens a file with (1h). More than 7 days refuses the boot: no bucket takes it. The server reads a file back over a link of its own, good for a minute
- `support.attachments.bucket.endpoint`, `region`, `name`, `accessKeyId` — the S3 endpoint (a scheme and a host), the region the links are signed for (`auto` on R2), the bucket, and the key's id, which is in every link and is not a secret. Any of them empty with attachments on refuses the boot
- `support.attachments.bucket.secretAccessKey` — `env://SUPPORT_BUCKET_SECRET` in `deploy/vps/backend.yaml`; empty with attachments on refuses the boot
