# Six bounded contexts, one process

- **`internal/planet/`** — the tile game: clicks, ownership, the map, the update stream.
- **`internal/chat/`** — the live chat: messages, identity, retention.
- **`internal/auth/`** — who a caller is and what it has to prove before it may click: Turnstile, accounts, the click token, and which accounts may operate. See [Auth](auth.md).
- **`internal/player/`** — what the game keeps about one account: the name it chose, the tiles it took, its daily streak. See [Player](player.md).
- **`internal/seasons/`** — the season calendar, each season's player standings and its daily rounds between countries, above the game. See [Seasons](seasons.md).
- **`internal/support/`** — what a player sends the operator: a feature suggestion, a bug report or a report of another player. See [Support](support.md).

Beside them, **`internal/clips/`** is a module of the server too, for one thing: it keeps the numbers of the clips
the clips job posts, for the operator alone. An admin service and its tables, no public route, no event, and nothing
it asks of another module. See [Clip stats](clip-stats.md).

They share the process, the transport and the country list, and **nothing else**. None imports another; each owns its own domain types, its own proto package, its own storage adapter (where it has one) and its own edge.

**A module never calls another module's code or reads its data in its own stack trace.** When one needs an answer from another it asks over Connect, on a loopback listener — see [Calling another module](composite-layer.md#calling-another-module). Five do: `planet`, `player`, `chat`, `seasons` and `support` take the click token's verifying key from `auth`, `player` asks `auth` whether an account is linked and `planet` which countries an account took tiles for and from, `chat` asks `player` who posts: the username, or the guest code, `seasons` asks `player` who has a username and `planet` what each country holds, and tells `planet` the rule set of each phase, and `support` asks `player` how to name the accounts of a report. When one only has to say that something happened, it publishes an event on Redis Streams and never learns who listens — see [Telling other modules what happened](composite-layer.md#telling-other-modules-what-happened). `planet` publishes `TileTaken`, `BombLanded`, `Fortified`, `BonusClaimed` and `QuizAnswered`, `auth` publishes `AccountDeleted`, `SignedIn` and `SignedOut`, `chat` publishes `MessageSent` and `AccountMuted`, `player` publishes `StatsChanged`, and `seasons` publishes `RoundClosed` and `SeasonClosed`; `player` hears every one of them but `AccountMuted`, `RoundClosed` and `SeasonClosed`, `chat` hears `AccountMuted`, `BombLanded`, `Fortified`, `RoundClosed` and `SeasonClosed`, `seasons` hears `TileTaken` and `AccountDeleted`, and `planet`, `chat` and `support` hear `AccountDeleted` too.

Bonus boxes live inside `internal/planet/` rather than beside it: what they grant
is click allowance, and what carries them is the planet stream. A context of
their own would have to import both.

**`internal/antibot/` is a domain library, not a fifth context.** It has no proto
package, no adapters and no `module.go`, and it cannot be wired without a caller
composing it — the clicks edge does, the way it gates on `auth`. It is not a
shared package because "is this caller a bot" is the business this game is in,
while `shared` is for things that would read the same in any other program.
`planet` → `antibot` is the only module-to-module import in the backend.

**`internal/clips/` is mostly a job, with a binary of its own** (`cmd/clips`), and a small module of the server
beside it. The job shares no process and no config with the server, and imports only `shared`: what it asks of the
server (a replay, the race, the season) and what it gives it (the snapshots of `stats`, which the clips module keeps)
goes over ssh to the console listener, as an operator's call. **One package, two roots**: `clips.Config` and
`Fetch`, `Choose`, `Schedule`, `Stats` are the job's, `clips.ModuleConfig` and `NewModule` the server's, and only
`NewModule` is wired into `cmd/api`. The clips are one concept, so their numbers live with them rather than in a
module of their own. See [The clips job](clips-job.md).

## A module publishes its root package and hides the rest

Every module's interior lives behind **its own `internal/`** — `internal/planet/internal/clicks`,
`internal/chat/internal/messages`, `internal/antibot/internal/jury`. Go's own
rule does the enforcing: such a package is importable only from the tree rooted
at the parent of that `internal`, so `chat` importing `planet/internal/clicks`
**does not compile**. There is no linter to run, no allowlist to maintain and
nothing to keep in sync.

So the whole of what one module may use of another is **what sits in the other's
root package**:

| module | its public API |
|---|---|
| `planet` | `Config`, `NewModule` |
| `chat` | `Config`, `StorageConfig`, `NewModule` |
| `auth` | `Config`, `NewModule`, `AdminGateConfig`, `NewAdminGate`; behind the `testing` tag, `NewModuleWithFakeProviders` and the fake's types |
| `player` | `Config`, `NewModule` |
| `seasons` | `Config`, `NewModule` |
| `support` | `Config`, `NewModule` |
| `antibot` | `Config`, `Observer`, `Guard`, `New`, `Description`, `Click`, `Report`, `Sentence`, `Pardon`, `Examination`, `Reading` |
| `clips` | `ModuleConfig`, `NewModule` (the server's); `Config`, `ReplayConfig`, `Fetch`, `Choose`, `Schedule`, `Stats`, `Pick`, `Platform`, `PlatformsOf`, `Report`, `PlatformOutcome`, `StatsReport`, `TikTokLane`, `TikTokAccount`, `Unread` |

That holds for `cmd/api` too: the composition root lists modules and cannot
reach a domain type, a storage adapter or a controller even if it wanted to. A
module's `Config` may carry a field whose *type* is internal (`planet.Config.TilesStorage`
is `inmemory_tile_storage.Config`) — koanf fills it by reflection and a caller can
still set its fields, it just cannot name the type. That is the right amount of
access: the settings are published because they are in the file, and the code
that reads them is not.

**Adding a package inside a module is therefore free, and taking one out of
`internal/` is a deliberate act** that shows up in review as exactly one moved
directory.
