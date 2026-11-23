# The flags module

**The flags module owns the catalog of flags**: the 239 the game ships with, and the flags opened since. It is step 1
of the regional flags plan (2026-11-23): a flag opened after the server starts is known everywhere with no deploy. No
flag is opened yet, and players see no change.

## The catalog

- **The flags the game ships with** are the page's `static/countries/countries.json`, copied into
  `generated/flags/countries.json` by `make flags` (`check-flags` in CI and in the pre-push hook) and embedded:
  `cpflags.BuiltIn()` (in the page's order, with each English name) and `cpflags.IsBuiltIn`. Never a list kept by hand:
  it replaced `shared/cpcountries` and its `data.go`.
- **A flag opened since** is a row of `flags.flags`: `id`, `name`, `state` (`flag_state`: `open` or `closed`) and
  `decided_at`, when it took that state. **A flag is closed, never deleted**: the season standings, the Hall of Fame,
  the map history and the videos still name it.
- **Its id is `catalog.FlagID`**: `r-` and ten letters or digits (`r-k3p9x2ma0b`), drawn at random. A flag the game ships
  with is two letters, or `gb-` and three letters, so no ISO code and no `gb-xxx` code can take one. Never the name: a
  review in step 2 may change the name, and the id stays.
- **Its name is `catalog.Name`**: 1 to 40 characters, trimmed, no control character, in English. It is not translated.

## What it tells

`flags.v1.FlagOpened{flag, name, opened_at}` and `flags.v1.FlagClosed{flag, closed_at}`, through its outbox (a lost one
would leave planet refusing a flag for good), in the transaction that opens or closes the flag.

**At each boot it tells every flag again** (`restate_flags_usecase`), as the antibot tells planet every running ban again
(`bans.Book.Restate`): an open flag as `FlagOpened`, a closed one as `FlagClosed` alone. A module that subscribes after a
flag was told hears it at the next boot, with no hand migration. With no flag opened, it tells nothing.

**Of one flag, the fact with the latest time holds.** The two facts go on two streams, so a listener may hear a close
before the opening it follows; it keeps the later one, and a fact told again changes nothing.

## Planet keeps a catalog, the others check a code's shape

**Planet alone keeps what it heard**, in its own schema (`planet.heard_flags`: `flag`, `state` as `heard_flag_state`,
`decided_at`), because it plays the flags: whether one is open is a rule of the click.

- **`catalog.Catalog`** (`internal/planet/internal/catalog`) answers two questions. `Playable`: a flag the game ships
  with, or one heard open. `Known`: the same, or one heard closed. A flag the game ships with is never read from
  postgres, so today's clicks cost nothing more.
- **`flag_opened_subscriber` and `flag_closed_subscriber`** hear the facts (`planet-flags-opened`,
  `planet-flags-closed`). `postgres_catalog_store` keeps one row per flag under an advisory lock, and only when a fact
  changes it.
- **Only the map lives in RAM**: the catalog reads postgres, for a flag that is not built in.
- **Not an RPC to the flags module**: a click for a flag opened since is one read of planet's own postgres, not a call
  on every click, as planet keeps the antibot's bans (`refused_callers`).

| asks | for |
|---|---|
| `Playable` | a click, a pledge (and the allegiance a click checks), a bomb, a shield, a virus, a mine, a rally |
| `Known` | the operator's `FindPlayers`, a flag and an area |

**The chat, scoring and player check a code's shape alone** (`cpflags.Codes`): a flag the game ships with, or `r-` and
ten letters or digits (`cpflags.IsOpenedCode`, which `catalog.ParseFlagID` reads too). They only show a flag or filter by
one: the flag a message is sent under, a board or a tally asked for one flag, the flag a player is announced under. A
made-up `r-` code there shows as its code with a grey flag on the page, and a board of it is empty. Their code is
`main`'s, with `cpflags.Codes{}` in place of `cpcountries.New()`.

**`shared/cpflags` holds what more than one module needs**: the flags the game ships with (planet, the flags module,
the alerts, the chat, scoring and player) and the shape of a code opened since.

**A closed flag is refused as an unknown one** (`clicks.ErrUnknownCountry`) for a click, a pledge and every charge. Its
land stays, shields too, and its players keep it as their flag. Step 2 builds what a close does to its players (pick
another flag) and to its chat room (read-only): planet's `flag_closed_subscriber` is where it goes, a use case in place
of the bare catalog.

## The stream

**The flags module tells the pages itself**, on a stream of its own: `flags.v1.FlagService/ListenForEvents`, a
`heartbeat` at once and every `httpServer.streamHeartbeat`, and `flag_changed`, each flag opened or closed while the
stream is open, as the catalog holds it now (`Flag{code, name, closed}`). It reads no token: every page hears the same.

- **The module hears its own facts** (`flags-streams-opened`, `flags-streams-closed`) and tells each to the open streams
  (`tell_streams_usecase`), the flag read from its table, so a close heard before its opening still tells the flag
  closed. A fact told again (the restatement at each boot) changes nothing for a page, so it is skipped: only a fact
  decided since this process started is news (`feed.News`). A page reads `ListFlags` each time its stream opens, so it
  misses nothing told while the stream was down.
- **A stream that falls 32 flags behind is closed, not skipped** (`feed.Board`), as the roster's: its page opens it
  again and reads the list.
- **A page holds one more stream for it.** A guest page held six before (planet, quizzes, boxes, player, gamemaster,
  chat): over HTTP/2 in production they share one connection, but a plain-HTTP/1.1 local stack has 6 connections per
  host, and its unary calls wait.

See the frontend's [flag catalog](../../frontend/docs/flag-catalog.md).

**The alerts keep the flags the game ships with.** A Discord role is made per built-in flag; a flag opened since has
none yet. `nations.FlagOf` answers no flag for it, so a pledge to it takes the user's role off, and a take by it or of
its land pings nobody. It is never an error that holds the alerts' stream. Roles for the flags opened since are a later
step.

## The reads

`flags.v1.FlagService/ListFlags`, public, and `flags.v1.AdminService/ListFlags`, for the backoffice and Claude
(`flags_list_flags`): every flag the game ships with, in the page's order, then the flags opened since, by name, the
closed ones with `closed`. One query (`flags_query`), no cache header: the page reads it again each time its stream
opens, and a cached answer would not hold a flag opened while it was down.

## Open and close

`open_flag_usecase` (a name in, a new id out) and `close_flag_usecase` (a flag closed once: `catalog.ErrClosed`) record
their fact in the transaction that keeps the flag. **In step 1 only tests call them**, so they, `catalog.OpenFlag` and
`random_flag_ids` are behind the `testing` tag: `make deadcode` refuses code with no caller. Step 2's operator call
drops the tags.

`internal/flags/told_test.go` opens a flag through the module with a real postgres and Redis, and watches another
module's catalog make it playable with no restart, then not playable once closed.

## The pool

The flags module holds 1 postgres connection; player went from 4 to 3, so `max_connections` stays 42.
