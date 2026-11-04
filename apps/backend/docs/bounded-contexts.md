# Eleven bounded contexts, one process

- **`internal/planet/`** — the tile game: clicks, ownership, the map, the update stream.
- **`internal/chat/`** — the live chat: messages, identity, retention.
- **`internal/auth/`** — who a caller is and what it has to prove before it may click: Turnstile, accounts, the click token, which accounts may operate, and how each account came. See [Auth](auth.md) and [Arrivals](arrivals.md).
- **`internal/player/`** — who an account is to the other players: the name it chose or its guest code, its color, who is playing now, and an account asked deleted. It tells the other modules what they need of an account, so none needs to hear auth. See [Player](player.md).
- **`internal/progression/`** — how far an account went: the tiles it took, its daily streak, what it did with the bonuses, its titles and the one it wears, the season finales it fought, the countries it fought for and against, and the badges drawn beside its name. It hears the whole game and nothing speaks its language but the browser and the operator. See [Progression](progression.md).
- **`internal/scoring/`** — the points accountant: the books and rounds the gamemaster opens and closes by id, each book's player tallies, and each round's results between countries. It knows no season, day or battle. See [Scoring](scoring.md).
- **`internal/support/`** — what a player sends the operator: a feature suggestion, a bug report or a report of another player. See [Support](support.md).
- **`internal/missions/`** — each account's four or five missions of the day, how far it took them, and what each pays once done; and each flag's mission of the day, which gives the flag season points. See [Missions](missions.md).
- **`internal/antibot/`** — whether a caller plays by hand: the watchdogs, the jury and the bans. It hears what callers do from planet and asks planet to refuse the ones it bans. See [Anti-bot](antibot.md).
- **`internal/gamemaster/`** — the game's meta: which rule sets planet, the antibot and scoring play together, and when, on a schedule of entries (a battle tonight, the Final Battle), and when each season, day and battle starts and ends, told to scoring by id. It knows the sets' names, never their values: never a point. See [The gamemaster module](gamemaster.md).
- **`internal/alerts/`** — the game's Discord server: a player's Discord link, which a Discord sign-in makes, the role of the flag it plays for, and the pings when that flag is attacked (its role) or a battle or a season starts (everybody). Off by default. See [Discord pings](alerts.md).

Beside them, **`internal/clips/`** is a module of the server too, for one thing: it keeps the numbers of the clips
the clips job posts, for the operator alone. An admin service and its tables, no public route, no event, and nothing
it asks of another module. See [Clip stats](clip-stats.md).

They share the process, the transport and the country list, and **nothing else**. None imports another; each owns its own domain types, its own proto package, its own storage adapter (where it has one) and its own edge.

**A module never calls another module's code or reads its data in its own stack trace.** When one needs an answer from another it asks over Connect, on a loopback listener — see [Calling another module](composite-layer.md#calling-another-module). `planet`, `player`, `progression`, `chat`, `scoring`, `support`, `missions`, `alerts` and `clips` take the click token's verifying key from `auth`. `player` asks `auth` whether an account is linked and when it was made, and answers the same of a page of accounts to whoever must not ask auth (`GetAccounts`). `progression` asks `player` that (`GetAccounts`), `planet` which countries an account took tiles for and from, and `scoring` a closed book's table and who fought a season's finale. `chat` asks `player` who posts: the username, or the guest code; `scoring` asks `player` who has a username and `planet` what each country holds; `support` asks `player` how to name the accounts of a report; and `missions` asks `planet` the flag an account pledged, what each flag holds and how big each country is, and `auth` whether an account is signed in. `gamemaster` asks `planet`, `antibot` and `scoring` which rule sets they keep. `alerts` asks `planet` the flag an account pledged, and `auth` the token a Discord sign-in left to join the game's server by (`GetDiscordGrant`); `auth` asks no other module anything. When one needs another to change something, it sends a command: `missions` commands `planet` to grant a mission's reward (`planet.v1.GrantCharges`) and `scoring` to grant a flag mission's points (`scoring.v1.GrantPoints`), `antibot` commands it to refuse or admit a caller (`planet.v1.RefuseCaller`, `AdmitCaller`), and `gamemaster` commands `planet` and `antibot` to play a rule set (`planet.v1.PlayRuleSet`, `antibot.v1.PlayRuleSet`), `planet` to reset the map as a season starts (`planet.v1.ResetMap`), and `scoring` to open and close its books and rounds (`scoring.v1.ScoringCommand`); `chat` asks `scoring` a round's results and a book's table by id. When one only has to say that something happened, it publishes an event on Redis Streams and never learns who listens — see [Telling other modules what happened](composite-layer.md#telling-other-modules-what-happened). `planet` publishes `TileTaken`, `AllegiancePledged`, `BombLanded`, `Fortified`, `ShieldPlaced`, `ShieldStruck`, `BonusClaimed` and `QuizAnswered`, and what callers did for whoever judges it (`ClickTried`, `MapRead`, `StreamOpened`, `BoxCaught`, `BoxLapsed`, `ForeignBoxClaimed`), `auth` publishes `AccountDeleted`, `AccountDeletionRequested`, `AccountRestored`, `SignedIn`, `SignedOut` and `DiscordJoinGranted`, `chat` publishes `MessageSent` and `AccountMuted`, `player` publishes `PlayerCreated`, `PlayerLeaving`, `PlayerReturned` and `PlayerForgotten`, `progression` publishes `StatsChanged`, `missions` publishes `MissionsPaid`, `scoring` publishes `RoundScored` and `BookClosed`, and `gamemaster` publishes `SeasonStarted`, `DayStarted`, `BattleStarted`, `DayEnded`, `BattleEnded` and `SeasonEnded`. `player` hears `AccountDeleted`, `AccountDeletionRequested`, `AccountRestored`, `SignedIn` and `SignedOut`, and nothing else; `progression` hears `TileTaken`, `BombLanded`, `Fortified`, `BonusClaimed`, `QuizAnswered`, `MessageSent`, `MissionsPaid`, `SeasonEnded`, player's four and its own `StatsChanged`, and nothing of auth's; `planet` hears `PlayerCreated` and `AccountDeleted`; `gamemaster` hears `RoundScored` and `BookClosed`, `chat` hears `AccountMuted`, `AllegiancePledged`, `BombLanded`, `Fortified`, `DayEnded`, `BattleEnded` and `SeasonEnded`, `scoring` and `missions` hear `TileTaken` and `AccountDeleted`, `missions` hears `SignedIn`, `BonusClaimed`, `QuizAnswered`, `BombLanded`, `ShieldPlaced`, `Fortified`, `ShieldStruck`, `DayStarted` and its own `MissionsPaid` too, and `chat`, `support` and `auth` itself hear `AccountDeleted` too; `antibot` hears planet's facts about callers; `alerts` hears `TileTaken`, `AllegiancePledged`, `AccountDeleted`, `DiscordJoinGranted`, `BattleStarted` and `SeasonStarted`.

**The modules depend on each other with no cycle.** A module depends on another when its code speaks the other's proto package: it calls it, hears its events or sends it commands. `cmd/api/modules_test.go` reads every module's imports and pins it: `TestTheModulesDependOnEachOtherWithNoCycle`, `TestNothingButTheBrowserAndTheOperatorTalksToProgression` and `TestProgressionKnowsPlayerAndNotAuth` (the session token's key, through `cpsessionverifier`, is the one way every module knows auth). Today: `auth` depends on none; `player` on `auth`; `planet` on `auth` and `player`; `antibot` on `planet`; `scoring` on `auth`, `planet` and `player`; `gamemaster` on `antibot`, `planet` and `scoring`; `missions` on `auth`, `gamemaster`, `planet` and `scoring`; `chat` on `auth`, `gamemaster`, `planet`, `player` and `scoring`; `support` on `auth` and `player`; `alerts` on `auth`, `gamemaster` and `planet`; `clips` on `auth`, `gamemaster`, `planet`, `player` and `scoring`; `progression` on `chat`, `gamemaster`, `missions`, `planet`, `player` and `scoring`. Player and progression were one module until 2026-11-04, and that module made three cycles: see [Why a module of its own](progression.md#why-a-module-of-its-own).

**A downstream module asks its upstream in the upstream's words.** `antibot` sends planet `RefuseCaller` and `AdmitCaller`, commands planet defines in its own proto package (`proto/planet/v1/commands.proto`) and consumes into its own table. Planet never hears the antibot's vocabulary: it tells what happened and is told whom to refuse. A command is an intent in the upstream's published language, not a fact, so it does not break the rule that facts flow from where they happen — see [Telling other modules what happened](composite-layer.md#telling-other-modules-what-happened).

Bonus boxes live inside `internal/planet/` rather than beside it: what they grant
is click allowance, and what carries them is the planet stream. A context of
their own would have to import both.

**`internal/antibot/` was a domain library until 2026-10-26**, built by planet's click chain and called on every click: `planet` → `antibot` was the only module-to-module import. It is a module now, wired in `cmd/api` like the others, and no module imports another.

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
| `progression` | `Config`, `NewModule` |
| `scoring` | `Config`, `NewModule` |
| `support` | `Config`, `NewModule` |
| `missions` | `Config`, `NewModule` |
| `antibot` | `Config`, `NewModule` |
| `gamemaster` | `Config`, `NewModule` |
| `alerts` | `Config`, `DiscordConfig`, `NewModule` |
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
