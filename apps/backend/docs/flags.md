# The flags module

**The flags module owns the catalog of flags**: the 238 the game ships with, and the flags sold since. Step 1 of the
regional flags plan (2026-11-23) made the list live: a flag opened after the server starts is known everywhere with no
deploy. Step 2 lets the operator open one by hand, with a picture and an end, add time to it and close it; it closes by
itself at its end, and comes back within 30 days. Since 2026-11-24 the operator renames one. The first real one opens
when the owner says so.

## The catalog

- **The flags the game ships with** are the page's `static/countries/countries.json`, copied into
  `generated/flags/countries.json` by `make flags` (`check-flags` in CI and in the pre-push hook) and embedded:
  `cpflags.BuiltIn()` (in the page's order, with each English name) and `cpflags.IsBuiltIn`. Never a list kept by hand:
  it replaced `shared/cpcountries` and its `data.go`.
- **A flag opened since** is a row of `flags.flags`: `id`, `name`, `state` (`flag_state`: `open` or `closed`),
  `decided_at`, when it took that state, `ends_at`, and its picture (`picture_hash`, `_width`, `_height`, `_stretch`,
  `_focus_x`, `_focus_y`). **A flag is closed, never deleted**: the season standings, the Hall of Fame, the map
  history and the videos still name it. A flag opened by mistake that nobody played for was deleted by hand in postgres
  (`flags.flags` and `planet.heard_flags`) on 2026-11-24; there is no call for it.
- **Its id is `catalog.FlagID`**: `r-` and ten letters or digits (`r-k3p9x2ma0b`), drawn at random. A flag the game ships
  with is two letters, or `gb-` and three letters, so no ISO code and no `gb-xxx` code can take one. Never the name: a
  review in step 2 may change the name, and the id stays. **A flag that left the build keeps the code it shipped with**
  (Brittany, `xb`, below), so its land, players, room and points stay its own.
- **Its name is `catalog.Name`**: 1 to 40 characters, trimmed, no control character, in English. It is not translated.
  **Two flags never share a name**, whatever its case, a flag the game ships with included (`catalog.Names`, at the open
  and at a rename), and `flags_name_key`, a unique index on `lower(name)`, holds it against two calls at once.

## What it tells

`flags.v1.FlagOpened{flag, name, opened_at}`, `FlagClosed{flag, closed_at}` and `FlagRenamed{flag, name, renamed_at}`,
through its outbox (a lost one would leave planet refusing a flag for good), in the transaction that changes the flag.
The outbox's `outbox_event` enum names each. Planet keeps no name, so only the flags module hears `FlagRenamed`.

**At each boot it tells every flag again** (`restate_flags_usecase`), as the antibot tells planet every running ban again
(`bans.Book.Restate`): an open flag as `FlagOpened`, with its name now, a closed one as `FlagClosed` alone. A module that
subscribes after a flag was told hears it at the next boot, with no hand migration. With no flag opened, it tells
nothing.

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
| `Playable` | a click, a pledge (and the allegiance a click checks), a bomb, a shield, a virus, a mine, a trap, a rally |
| `Known` | the operator's `FindPlayers`, a flag and an area |

**The chat, scoring and player check a code's shape alone** (`cpflags.Codes`): two letters or `gb-` and three, as the
flags the game ships with, or `r-` and ten letters or digits. A code of that shape the game does not ship is a sold
flag's (`cpflags.IsOpenedCode`, which `catalog.ParseFlagID` reads too). They only show a flag or filter by one: the flag
a message is sent under, a board or a tally asked for one flag, the flag a player is announced under. A made-up code
there shows as its code with a grey flag on the page, and a board of it is empty. Their code is
`main`'s, with `cpflags.Codes{}` in place of `cpcountries.New()`.

**`shared/cpflags` holds what more than one module needs**: the flags the game ships with (planet, the flags module,
the alerts, the chat, scoring and player) and the shape of a code opened since.

**A closed flag is refused as an unknown one** (`clicks.ErrUnknownCountry`) for a click, a pledge and every charge. Its
land stays, shields too: the game never removes land by itself, and rivals take it over time. Its season points stay
(scoring counts what was taken).

- **Its players pick another flag at once, with no hour of wait.** Planet's `allegiances.Book` stands a flag only if the
  arena keeps it in and the catalog finds it playable (`standingFunc`), so an account whose flag closed changes as one
  whose flag was knocked out does (`Allegiance.Pledged`). Nothing is written at the close: the next pledge does it.
  `GetAllegiance` and `GetAllegianceOf` answer `Allegiance.closed`, and the page asks for another flag.
- **While the rules in force lock the flags (a battle), a closed flag stays playable** until the lock lifts: the
  catalog's `Playable` is open, or closed while `lock()` holds (`catalog.NewCatalog(store, lock)`, planet's
  `allegiances.LockPart`). The flags module closes at the end date and knows no battle; the lock is planet's.
- **Its chat room is read, not written**: the chat asks planet `GetAllegianceOf` for a flag's room already, and a
  message or a reaction in the room of a closed flag is `messages.ErrRoomClosed` (`FailedPrecondition`,
  `Rooms.SpeakError`). The world is no flag's room.

## The stream

**The flags module tells the pages itself**, on a stream of its own: `flags.v1.FlagService/ListenForEvents`, a
`heartbeat` at once and every `httpServer.streamHeartbeat`, `flag_changed`, each flag opened, closed or renamed while the
stream is open, as the catalog holds it now (`Flag{code, name, closed, sold}`). It reads no token: every page hears the
same.

- **The module hears its own facts** (`flags-streams-opened`, `flags-streams-closed`, `flags-streams-renamed`) and
  tells each to the open streams (`tell_streams_usecase`), the flag read from its table, so a close heard before its opening still tells the flag
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
closed ones with `closed`. A flag opened since carries `sold`: its picture (public URL, size, `stretch` and focus), its
`ends_at` and, once closed, `closed_at`. One query (`flags_query`), no cache header: the page reads it again each time
its stream opens, and a cached answer would not hold a flag opened while it was down.

## The operator's calls

`flags.v1.AdminService`, on the backoffice's Flags page and as Claude's tools (`flags_create_flag_upload`,
`flags_open_flag`, `flags_add_time`, `flags_close_flag`, `flags_rename_flag`). **Every player sees a flag opened at
once, and a closed flag is never deleted: open one only when the owner says so.** Each change reads the flag
`FOR UPDATE` in its transaction, so two changes at once never undo each other (a rename as the runner closes the flag
at its end keeps it closed).

- **`CreateFlagUpload(content_type, size_bytes)`**: a one-time link, good for 10 minutes, to `PUT` a PNG, JPEG or WebP
  of 5 MB at most into the bucket, under `flags/uploads/<id>`, private.
- **`OpenFlag(name, upload_id, ends_at)`** (`open_flag_usecase`): the picture drawn again (below), made public, then the
  flag kept and `FlagOpened` recorded in one transaction; the upload deleted once the flag is kept, so a refused open (an
  end in the past, a name taken) can be tried again with the same file. `catalog.Opening` draws the id and checks the
  end is ahead.
- **`AddTime(code, days)`** (`add_time_usecase`, 1 to 3,660 days): an open flag ends that much later, and tells nobody.
  **A flag closed less than 30 days ago comes back** (`Flag.TimeAdded`): the same code, name and picture, the land it
  still holds, open from now until that many days from now, told as `FlagOpened`. Past the 30 days,
  `catalog.ErrTooLate` (`FailedPrecondition`): the place is free for a new flag.
- **`CloseFlag(code)`** (`close_flag_usecase`): now, as a flag that broke the rules. `catalog.ErrClosed` for one closed
  already.
- **`RenameFlag(code, name)`** (`rename_flag_usecase`): open or closed; `catalog.ErrNameTaken` or `ErrInvalidName`
  (`InvalidArgument`) for a name another flag has or no name. Told as `FlagRenamed`: the pages show it at once, and the
  code, the picture, the land and the players stay.
- **At its end a flag closes by itself**: `expire_flags_usecase`, every 30 seconds, closes each open flag whose
  `ends_at` passed (`Store.Ending`, an index on `ends_at` of the open ones), at its end, and records `FlagClosed`.

## The pictures

**Each flag opened since wears a picture of its own, never in the atlas**: no deploy per flag, and pages load only the
pictures of the flags there are.

- **Drawn again by the server** (`pictures.Redraw`): only a PNG, a JPEG or a WebP, read by its first bytes, whatever
  type the upload said (an SVG, which can carry a script, is refused, and so is a GIF); no animation (an APNG's `acTL`,
  an animated WebP's flag); between square and 2 wide by 1 high, at least 100 wide; fitted inside 768 x 512 (the big
  flags of the far view) with Catmull-Rom; written out as a PNG, which drops every chunk but the pixels (names, places,
  dates the file held).
- **Named after its content** (`flags/pictures/<sha256>.png`), written with `Cache-Control: public, max-age=31536000,
  immutable`: a name never holds another picture. A picture drawn twice is the same file.
- **Its fit** (`pictures.Fit`): what `scripts/generateFlagFit.mjs` gives a flag the game ships with, worked out the same
  way on a copy 100 wide, as the atlas's sprites are: `stretch` for a flag of bands only, and the focus a crop keeps.
- **Public on `https://media.clickplanet.lol`**, beside promo's backgrounds, with the support bucket's key
  (`flags.pictures` in `backend.yaml`). The zone must let that path through and add `Access-Control-Allow-Origin: *`
  (WebGL reads a picture with CORS, and the cache keeps one copy whatever the origin): see `deploy/vps/README.md`,
  "Sold flags' pictures". **Off** (`flags.pictures.enabled: false`, the example config), every picture call is
  `FailedPrecondition`, and no flag can be opened.
- `shared/cps3` reads (`Read`, with a size limit) and writes (`Write`, with its cache header) through signed links.

`internal/flags/told_test.go` opens a flag through the module with a real postgres and Redis, and shows the fact reach
another module's subscriber and an open page's stream with no restart, then the close and a rename.

## Brittany, from the build to the sold flags

**Brittany shipped with the game as `xb` until 2026-11-24.** It is a region, so the owner wants it sold and managed
like the others. One deploy moves it, and it keeps its code, so nothing of it is rewritten: its land and shields, its
players' pledges, its chat room, its season points, its ledger rows.

- **The build no longer lists it**: the page's `countries.json` (so `generated/flags/countries.json`, `make flags`), its
  sprite in the atlas (`npm run atlas`), its SVG for the far view's slots, its fit (`npm run flagFit`), its names in the
  page's languages (dropped by `cmd/translations apply`), the backoffice's and the alerts' own names for it.
- **The flags module holds it from the first boot of that build** (`20261124083424_brittany_sold`): open, named
  "Brittany", ending 30 days after the deploy, its picture the build's `xb.svg` (rendered at its 1350 x 900 with
  `rsvg-convert`) as `pictures.Redraw` draws an upload, `flags/pictures/ed47508c…3c3eb6.png`, 768 x 512. The restatement
  at boot tells it to planet.
- **Planet hears it open in the same deploy** (`20261124083425_brittany_heard`): the restatement travels through the
  outbox and Redis, and a page reloads within a second of the restart, so without it a Brittany player could be told its
  flag closed and asked for another.
- **Its picture goes into the bucket by hand, once**, since no flag was opened through `OpenFlag`: until it does, the
  page draws Brittany as the blank.
- **What it lost**: its Discord role (the alerts' tidy at boot deletes a role whose flag the game does not ship; sold
  flags get roles in a later step), its names in other languages (a sold flag's name is English), its old address
  (`clickplanet.lol/xb/<name>`). Its 16 quiz questions moved to subject `fr`, their ids kept, so their ratings stay: the
  quiz generator takes only a subject the game ships.

Production, in order, each with the owner's yes:

1. Put the picture in the bucket: `clickplanet-media`, key `flags/pictures/ed47508c67e67af2c0711725441736fbe62b614fb9b9c587a05efc51023c3eb6.png`,
   `Content-Type: image/png`, `Cache-Control: public, max-age=31536000, immutable`. The file is on the PR's assets branch.
2. Deploy: frontend first, then the backend (see [A server restart reloads the page](../../../docs/reload-on-restart.md)).
3. Then the owner's call: keep it until a date (`flags_add_time`, `xb`, the days past its 30), or close it now
   (`flags_close_flag`, `xb`); within 30 days of a close, `flags_add_time` brings it back.

## The pool

The flags module holds 1 postgres connection; player went from 4 to 3, so `max_connections` stays 42.
