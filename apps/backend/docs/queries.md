# Reads are queries

**A read endpoint does not go through the write model.** Domain types are built for the rules a write checks, and
a read that reuses them forces public fields onto them for its own sake. So a read is a query of its own. Chat
(`GetHistory`, `GetMessage`, `ListMessages`), player (`GetProfile`, `GetStats`, `GetPlayer`, `GetTitles`, `GetAuthors`, `GetRoster`), planet
(`GetTakesByCountry`, `GetTerritories`, `GetLedger`, `GetTileHolder`, `GetLandmass`), auth
(`GetMe`, `GetAccount`, `GetAccounts`), seasons (`GetStandings`, `GetMySeason`, the live race and `GetRace`) and support (`ListNotes`, `ListReports`) follow this so far. **A command may
still answer a small struct** (`SetName`, `WearTitle`, `React`, `GetAuthor`, which draws a guest code,
`signin.Admission`); the rule is for what only reads. Auth's `GetSignInOptions` and `GetVerifyingKey` already read
through ports of plain values (`Names()`, `PublicKey()`) and stay as they are.

- **One package per read, under the handler that serves it**: `<controller>/<procedure>_handler/<read>_query/`. No
  `queries/` directory: a query has one caller, so it sits with it. `query_postgres.go` holds its one entry point,
  `PostgresQuery`. Its method is a builder, named for what it answers (`History`, `Player`). A read of memory
  rather than postgres is `query_memory.go` and `MemoryQuery` (`roster_query`).
- **SQL answers the wire's message.** The query projects straight into the proto response: ordering, windows,
  counts, `COALESCE`, `json_agg` and lateral joins in SQL; Go only decodes, checks and stitches in what SQL cannot
  reach.
- **Strict at the boundary.** A stored value the wire cannot carry is an error, never a default:
  `history_query.ErrUnknownReaction`, `history_query.ErrUnknownKind`, `playerread.ErrUnknownColor`,
  `me_query.ErrUnknownProvider`. A rollback past a new reaction, kind, color or provider therefore refuses the read
  until those rows are gone. An announcement's kind and an identity's provider are enum types, so postgres already
  refuses a value no migration added; the check in Go is for a value the type holds and this build does not. A title id the catalog
  no longer has is the exception: the catalog drops it from what is shown, and the reconciliation revokes it.
- **What SQL cannot read is a port in the wire's terms.** `history_query.Authors` answers `*player.v1.Author`, and its
  adapter is a subpackage (`history_query/rpc_player_authors`). The query asks it once and stitches the answer on.
  An adapter the commands already use is reused rather than copied: `player_query.Accounts` is
  `players/rpc_account_reader`. **The adapter is the only bridge to the domain**: the titles catalog is game rules as
  objects, so `GetPlayer`, `GetTitles` and `GetAuthors` reach it through a `Titles` port that takes the ids SQL read
  and answers `*player.v1.Title`, implemented by `playerv1controller/inprocess_title_catalog`; the roster is state in
  memory, reached through `roster_query.Lines`, implemented by `roster_query/inmemory_roster`.
- **Independent parts run at once** (`errgroup`): the messages and their authors, and the announcements.
- **The handler only maps**: the caller off the context, the query, the header. It declares the query as its port.
  What several queries share and that knows only the wire is in `playerv1controller/playerread` (`KeptColor`,
  `Career`); `playermessage` encodes domain values, so only the commands and the adapters use it.
- **It reads the write side's tables**, so its tests seed through the real stores (`postgres_message_store`,
  `postgres_reaction_store`, `postgres_announcement_store`) and assert the proto it answers. A store's own contract
  suite covers only what the write side reads back.
- **A query imports no domain package.** Player's queries import neither `players`, `titles`, `wearing`, `presence`
  nor `playermessage`; their ids are the shared `cpsession.AccountID`. A small display rule a command also applies
  is written again in the query's SQL, and each side has its own tests: a streak as of today (`Streak.AsOf`), the
  guest name (`DisplayNameOf`), and a guest showing no streak and no title (`Author.Shown`, `wearing.AuthorOf`). A
  game rule is never written again: titles go through the port above, never thresholds in SQL. Chat's
  `history_query` imports neither `messages`, `reactions`, `announcements` nor `feed`: it takes the
  window as `chat.storage.historySize` and `retention`, keeps the kinds it can carry (`kinds`), and reads a reactor
  as `account:<uuid>` in SQL, all of which the write side holds again (`messages.Window`, `announcements.Kinds`,
  `reactions.ReactorOf`). Auth's queries import neither `accounts` nor `signin`: `me_query` reads the `cp_sid`
  cookie, hashes it and asks SQL for a session not expired as of now, which the commands do with `accounts.Caller`
  (`CookieValue`, `TokenOf`, `Session.ExpiryError`); it lists the providers oldest link first, as the store does, and
  names them with a map of its own, as `authprovider` does. The handlers still parse an id with
  `accounts.AccountIDOf`, as every edge does.
- **Every rule written twice has a parity test**, which runs the same stored rows through both copies and asks for
  the same answer. Player: a command's stats as of a day against `GetStats`, and `GetAuthor` against `GetAuthors`.
  Chat (`history_query/parity_test.go`): a message is in the
  history exactly when `React` may react to it and `SendMessage` may reply to it; `React`'s answer against the
  history's reactions, cap of names included; `SendMessage`'s answer against the message the history reads back,
  and a reply's against its quote there; every `announcements.Kinds` the
  chat announces read back. Auth: `GetMe` against `accounts.Caller` and `Store.Account` over cookies and clocks up to
  a nanosecond either side of an expiry, the session's `Linked` included (`me_query/parity_test.go`); every provider
  the proto names, saved under the name `authprovider` gives it, read back as itself; `GetAccounts` against
  `GetAccount` and against the linked flag the click token carries (`accounts_query/parity_test.go`). A rule a parity
  test does not cover is a rule that can drift.
- **So the domain has no exported fields.** A player domain type is built by a constructor (`NewProfile`,
  `NewStats`, `NewVisit` to start one; `StatsOf`, `StreakOf`, `AccountOf` to restore one a store or
  another module kept), read through nouns (`Stats.TilesTaken()`, `Visit.Author()`), and changed
  through builders that answer a copy (`Stats.WithTake`, `Profile.Renamed`, `Author.Renamed`, `wearing.Author.Wearing`,
  `Visit.Keyed`). `players.Profile` has no getter: a store and the wire read it through `ProfileSnapshot`
  (`TakeSnapshot`, `Restore`), as chat's `Message` below. Chat the same: `NewRecord`, `NewWindow`, `AuthorOf`,
  `TitleOf`, `RankOf`, `reactions.On` and `Off` for a `Change`, `Tally.Named`, `NewAnnouncement`, `BombOf`, and
  `feed.MessageSent`, `ReactionsChanged` and `Announced` for the three cases of an `Update`. A test outside the
  package compares through those nouns, or through a view of them (`title_test.go`'s `standingView`), since it
  cannot build the value by hand; where a test must, a constructor behind the `testing` tag builds it
  (`reactions.CountOf`, `TallyFor`). Auth the same: `AccountOf`, `NewIdentity` and `IdentityOf`, `ClaimOf`,
  `GuestSession`, `LinkedSession` and `SessionOf`, `NewSignIn` with `WithNewAccount`, `WithIdentity` and
  `WithReplaced`, `NewFlow` and `FlowOf`, `NewChallenge` and `ChallengeOf`, `NewOffer`, and `signin.AdmissionOf` behind
  the tag. A config block keeps its exported fields, since koanf fills it (`accounts.Lifetime`, `signin.Config`,
  `signin.Client`). **A value a browser holds keeps its format**: `aes_flow_sealer` seals a flow and a challenge
  through structs of its own whose keys are the old exported fields, and `format_test.go` opens a cookie sealed in
  that JSON and seals the same JSON back.
- **A message has no getter: it is read through a snapshot.** `messages.Snapshot` is a plain struct that mirrors
  the fields of a `Message`, with no behaviour. `Snapshot.Restore()` is the one way to build a message and the one
  place its invariant is checked (it has an id): `Drafts` and the store both go through it. `Message.TakeSnapshot()`
  is the one way to read one: the stores write from it, `chatmessage.Encode` and `publishing_send_message` build the
  wire from it, and a test compares it. What a message does is a method on it (`Named`, `Answering`), so a use case
  tells it and never asks it. `Record.TakeSnapshot()` is the same for the row a store writes; nothing reads a
  record back, so it has no `Restore`. Tests build a message with `messages.Restored(t, snapshot)`, behind the tag.
  **Only `Message` and `Record` follow this so far**: `Author`, `Title`, `Rank` and the other concepts still read
  through nouns.
