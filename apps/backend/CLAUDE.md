# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

A Go backend for a collaborative map-clicking game, in **hexagonal architecture (ports & adapters)**. This file
is only an index (see the root [Docs rule](../../CLAUDE.md#docs-rule)): each line names a doc in [`docs/`](docs)
and says what is in it. Read the doc before you change its area.

## Working on it

- [Commands](docs/commands.md) — `make test` (Docker for the postgres and Redis tests), `-race`, `make deadcode`,
  `make lint`, `make tidy`, `make check-format`, `go run ./cmd/api`, `make proto`, the vendoring targets (`quiz`,
  `map`, `vpn-lists`, `disposable-domains`, `zernio`), `make dBuild`, and `cmd/clips`.
- [Configuration](docs/configuration.md) — `cpconfigs` loads YAML and environment with koanf; the full schema.
- [Testing](docs/testing.md) — testify; which suites need Docker (postgres, S3, Redis); `e2e/` holds smoke tests
  only, rules go in lower tests; the `testing` tag.
- [Linting](docs/linting.md) — golangci-lint at a pinned version, gofumpt, `.golangci.yaml`, and the linters left
  off on purpose.
- [Git hooks](docs/git-hooks.md) — what the repo's hooks run for this app.
- [Protobuf](docs/protobuf.md) — the contracts in the shared `/proto`, generated into `generated/proto`.

## How the code is shaped

- [Eight bounded contexts, one process](docs/bounded-contexts.md) — planet, chat, auth, player, seasons, support,
  missions, antibot.
  A module publishes its root package and hides the rest.
- [The composite layer](docs/composite-layer.md) — each context wires itself in its `module.go`; how a module
  calls another one, how it tells the others what happened and asks one to change something: Redis Streams,
  the idempotent receivers, the outboxes, the commands.
- [Inside the planet module](docs/planet-module.md) — one directory per concept, no `domain` or `adapters` layer;
  use cases.
- [Inside the chat module](docs/chat-module.md) — the same shape, for chat's seven concepts.
- [Reads are queries](docs/queries.md) — a read endpoint does not go through the write model.
- [Adapters](docs/adapters.md) — the primary (Connect handlers) and secondary (stores) adapters.
- [Transactions](docs/transactions.md) — a use case opens one with `InTx` and records its events in it; a store
  reaches postgres only through `db.On(ctx)`.
- [Every id is a type of its own](docs/ids.md) — never a bare `uuid.UUID`, `string` or `[]byte` in a signature.
- [Manipulators are verbs, builders are nouns](docs/verbs-and-nouns.md) — a name says if calling it changes
  anything.
- [A command's `run` is a table of contents](docs/command-run.md) — a CLI `main` is named stages, and `run` only
  wires them.
- [Shared](docs/shared.md) — the `internal/shared/` `cp*` infrastructure packages.
- [The error net](docs/error-net.md) — no handler's raw error reaches the wire.
- [Shared interceptors](docs/interceptors.md) — the `shared/cpconnect` interceptors that all contexts use.

## The tile game (planet)

- [Key flow](docs/key-flow.md) — the path of one click, step by step, from the mint to the stream.
- [The map load](docs/map-load.md) — `GetMap` is a cached HTTP GET; the stream carries what follows.
- [The live streams](docs/live-streams.md) — the `ListenForEvents` streams, served two ways, and how they end on
  shutdown.
- [Durability](docs/durability.md) — the map lives in memory and goes to postgres every second; what a crash
  loses.
- [Map geography](docs/map-geography.md) — tile adjacency from the shared map blob, checked at boot.
- [Rate limiting](docs/rate-limiting.md) — two buckets per click, what the answer says is left, and a big country
  refills slower.
- [VPN blocklist](docs/vpn-blocklist.md) — `Click` refused from vendored VPN ranges, before the rate limiter.
- [Allegiance](docs/allegiance.md) — an account plays for one flag, and may change it once an hour.
- [Rules](docs/rules.md) — the rule sets kept in postgres, the season's baseline and the operator's override over it.
- [Shields](docs/shields.md) — a charge on a tile; each shield takes one foreign click.
- [Fortify](docs/fortify.md) — a take that makes a landmass whole gives every tile of it a shield.
- [Who holds the tiles](docs/tile-holders.md) — each tile keeps the account whose take holds it; `GetLandmass`
  reads a landmass's holders and its last fortify.
- [Charges](docs/charges.md) — every bonus is a charge, kept until the player uses it.
- [Spread](docs/spread.md), [Bomb](docs/bomb.md), [Enclose](docs/enclose.md), [Refill](docs/refill.md) — what
  each charge does to a click or to the bucket.
- [Quizzes](docs/quizzes.md) — the quiz banner, its short clock, and the charge a right answer gives.
- [A newcomer's first box and first quiz](docs/welcome-bonus.md) — a new account gets both early, once.

## Who a caller is

- [Sessions](docs/sessions.md) — the click token that makes clicking cost something; the click token and the
  identity token.
- [Auth](docs/auth.md) — Turnstile, guest and signed-in accounts (Google, Discord, emailed code), operators,
  sign-out, deletion and pruning.
- [Anti-bot](docs/antibot.md) — a module that hears planet's facts about callers, judges them, and asks planet to
  refuse the ones it bans; what survives a restart, and the parts that are easy to get wrong.

## The other contexts

- [Chat](docs/chat.md) — messages, announcements, mutes, removals, replies, reactions, and what each account saw.
- [Player](docs/player.md) — names and guest codes, stats, streaks, titles, and who is playing now.
- [Seasons](docs/seasons.md) — the season calendar, its standings, the daily rounds and the finale.
- [Missions](docs/missions.md) — three missions a day per flag, counted from `TileTaken`, paid once done with a
  `GrantCharges` command to planet; why progress is not read from the ledger.
- [Support](docs/support.md) — suggestions, bug reports and player reports, with their photos and videos.
- [Clip stats](docs/clip-stats.md) — where the posted clips' daily numbers are kept, for the backoffice.

## Operator tools and jobs

- [Operator tools (`AdminService`)](docs/admin.md) — the console and admin listeners; `GrantCharges`,
  `GetReplay`, `GetLedger`, `PaintRandomTiles`, manual bans.
- [The clips job](docs/clips-job.md) — `cmd/clips`: every 2 hours, picks the best clips and posts them; its
  experiments, the model editor and the stats.
