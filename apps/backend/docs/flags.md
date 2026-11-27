# The flags module

**The flags module owns the catalog of flags**: the 235 the game ships with, and the flags sold since. Step 1 of the
regional flags plan (2026-11-23) made the list live: a flag opened after the server starts is known everywhere with no
deploy. Step 2 lets the operator open one by hand, with a picture and an end, add time to it and close it; it closes by
itself at its end. Since 2026-11-24 the operator renames one. Step 3 (2026-11-24) lets a
signed-in player submit a flag, and the operator approve or reject it ([Submissions](#submissions)). Step 4
(2026-11-24) lets its founder pay: the shop tells each payment's time, and the first opens the flag
([Paid time](#paid-time)). Since 2026-11-26 **a flag is held by whoever pays for it** ([Holders and claims](#holders-and-claims)):
once its time ends, any signed-in player may pay for it and hold it, and the operator's close is a removal, for good.

## The catalog

- **The flags the game ships with** are the page's `static/countries/countries.json`, copied into
  `generated/flags/countries.json` by `make flags` (`check-flags` in CI and in the pre-push hook) and embedded:
  `cpflags.BuiltIn()` (in the page's order, with each English name) and `cpflags.IsBuiltIn`. Never a list kept by hand:
  it replaced `shared/cpcountries` and its `data.go`.
- **A flag opened since** is a row of `flags.flags`: `id`, `name`, `state` (`flag_state`: `open`, `closed` at its end
  or by a refund, or `removed` by the operator), `decided_at`, when it took that state, `ends_at`, its picture
  (`picture_hash`, `_width`, `_height`, `_stretch`, `_focus_x`, `_focus_y`) and its pin (`pin_latitude`,
  `pin_longitude`). Its holder is a row of `flags.holders` ([Holders and claims](#holders-and-claims)). **A flag is
  closed, never deleted**: the season
  standings, the Hall of Fame, the map history and the videos still name it. A flag opened by mistake that nobody played
  for was deleted by hand in postgres (`flags.flags` and `planet.heard_flags`) on 2026-11-24; there is no call for it.
- **Its id is `catalog.FlagID`**: `r-` and ten letters or digits (`r-k3p9x2ma0b`), drawn at random. A flag the game ships
  with is two letters, so no ISO code can take one. Never the name: a review in step 2 may change the name, and the id
  stays. **A flag that left the build keeps the code it shipped with** (Brittany, `xb`; England, Scotland and Wales,
  `gb-eng`, `gb-sct`, `gb-wls`; below), so its land, players, room and points stay its own.
- **Its name is `catalog.Name`**: 1 to 40 characters, trimmed, no control character, in English. It is not translated.
  **Two flags never share a name**, whatever its case, a flag the game ships with included (`catalog.Names`, at the open,
  at a rename, at a submission's approval and opening, and at a removed flag's return), and `flags_name_key`, a unique
  index on `lower(name)`, holds it against two calls at once. **A removed flag's name is free** for a new flag: the
  names and the index (`WHERE state <> 'removed'`) count the open and the closed flags alone. A closed flag keeps its
  name: a player may claim it.
- **Its capital is `catalog.Pin`**: a latitude from -90 to 90 and a longitude from -180 to 180, in degrees. It is where
  the flag will start in Capital Siege (step 6), as a country starts on its capital; the pages and the backoffice call
  it the capital, the code and the wire `pin`. **A point, never a tile**: each board
  numbers its tiles its own way, and a new board renumbers them all. The operator gives it at `OpenFlag`, a submission
  carries it. Planet seats the flag on it at each Capital Siege's start (see the [seats](capitals.md#the-seats)). The
  game's own flags have no pin: each has ground of its own since England, Scotland and Wales, the last that had none,
  moved to the sold flags with their pins (below). The migration that brought the pins
  (`20261124101101_flag_submissions`) gave production's test flag of Corsica Ajaccio's pin and Brittany Rennes', by
  name, and stops on any other flag, so none is left with no pin.

## What it tells

`flags.v1.FlagOpened{flag, name, opened_at, pin}`, `FlagClosed{flag, closed_at}` and `FlagRenamed{flag, name,
renamed_at}`, and of a submission's verdict `SubmissionApproved{submission_id, approved_at}` and
`SubmissionRejected{submission_id, rejected_at}`, of a payment refused `TimeRefused{grant_id, refused_at}`
([Paid time](#paid-time)), and of a flag that waits on a person `ReviewAsked{submission_id, name, cause, why,
asked_at}` ([Ask a person](#ask-a-person)), through its outbox (a lost one would leave planet refusing a flag for good), in the transaction that changes the flag.
The outbox's `outbox_event` enum names each. Planet keeps no name, so only the flags module hears `FlagRenamed`, the
operator's rename: the pages read the flag from the table as it is now. `FlagOpened` carries the capital, at each
opening, claim, return and restatement. **No fact moves a capital** since 2026-11-24: `FlagMoved` told a founder's change
approved, and went with it (`20261124145305_no_changes` deleted any left in the outbox; the enum keeps its value, as
postgres drops none).

**At each boot it tells every flag again** (`restate_flags_usecase`), as the antibot tells planet every running ban again
(`bans.Book.Restate`): an open flag as `FlagOpened`, with its name now, a closed or removed one as `FlagClosed` alone.
**A removal is `FlagClosed`**, told again at its time for a flag closed already, so the pages hear it removed; planet
keeps it closed. A module that
subscribes after a flag was told hears it at the next boot, with no hand migration. With no flag opened, it tells
nothing.

**Of one flag, the fact with the latest time holds.** The two facts go on two streams, so a listener may hear a close
before the opening it follows; it keeps the later one, and a fact told again changes nothing.

## Planet keeps a catalog, the others check a code's shape

**Planet alone keeps what it heard**, in its own schema (`planet.heard_flags`: `flag`, `state` as `heard_flag_state`,
`decided_at`, and the pin: `pin_latitude`, `pin_longitude`, `pinned_at`, all null until a fact names one), because it
plays the flags: whether one is open is a rule of the click, and where it starts a siege a rule of the arena.

- **Of one flag's pin, the latest fact that named one holds** (`Heard.Moved`): `FlagOpened`'s at its `opened_at`. A
  pin told again at the time of the one kept replaces it, so the restatement at each
  boot puts a wrong pin right, and gave the flags opened before pins their first one (Brittany, Corsica) at the deploy.
  A flag never heard keeps no pin. A `FlagOpened` from a build before pins names none and keeps the pin as it is.

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

**The chat, scoring and player check a code's shape alone** (`cpflags.Codes`): two letters, as the flags the game ships
with, `gb-` and three, as England, Scotland and Wales shipped, or `r-` and ten letters or digits. A code of that shape
the game does not ship is a sold flag's (`cpflags.IsOpenedCode`, which `catalog.ParseFlagID` reads too). They only show a flag or filter by one: the flag
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
`heartbeat` at once and every `httpServer.streamHeartbeat`, `flag_changed`, each flag opened, closed, removed or
renamed while the stream is open, as the catalog holds it now (`Flag{code, name, closed, sold}`, `Sold.removed` for a
removed one). It reads no token: every page hears the same, never a flag's holder.

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
`ends_at`, its `pin`, once closed `closed_at`, and `removed` for one the operator removed: a closed flag not removed is
free to claim. One query (`flags_query`), no cache header: the page reads it again each time
its stream opens, and a cached answer would not hold a flag opened while it was down.

## The operator's calls

`flags.v1.AdminService`, on the backoffice's Flags page and as Claude's tools (`flags_create_flag_upload`,
`flags_open_flag`, `flags_add_time`, `flags_close_flag`, `flags_rename_flag`). **Every player sees a flag opened at
once, and a closed flag is never deleted: open one only when the owner says so.** A flag opened by hand has no holder:
nobody pays for it until its time ends, then any player may claim it. Each change reads the flag
`FOR UPDATE` in its transaction, so two changes at once never undo each other (a rename as the runner closes the flag
at its end keeps it closed).

- **`CreateFlagUpload(content_type, size_bytes)`**: a one-time link, good for 10 minutes, to `PUT` a PNG, JPEG or WebP
  of 5 MB at most into the bucket, under `flags/uploads/<id>`, private.
- **`OpenFlag(name, upload_id, ends_at, pin)`** (`open_flag_usecase`): the picture drawn again (below), made public, then the
  flag kept and `FlagOpened` recorded in one transaction; the upload deleted once the flag is kept, so a refused open (an
  end in the past, a name taken) can be tried again with the same file. `catalog.Opening` draws the id and checks the
  end is ahead.
- **`AddTime(code, days)`** (`add_time_usecase`, 1 to 3,660 days): an open flag ends that much later, and tells nobody.
  **A closed or removed flag comes back** (`Flag.TimeAdded`), however long ago: the same code, name, picture and
  holder, the land it still holds, open from now until that many days from now, told as `FlagOpened`; a removed one's
  note goes. **A removed flag whose name a new flag took since** is `catalog.ErrNameTaken` (`FailedPrecondition`):
  rename one of them first.
- **`CloseFlag(code, note)`** (`close_flag_usecase`): **a removal, for good** (the owner, 2026-11-26), labelled Remove on
  the backoffice and in the tool's description: now, as a flag that broke the rules, open or closed at its end, **with
  a note its holder reads** (the owner, 2026-11-24: a free note, no list of reasons; every removal, a flag with no
  holder too). Nobody pays for a removed flag (`catalog.ErrRemoved`), and its name is free for a new flag; only
  `AddTime` brings it back. `catalog.Note`: 1 to 500 characters once the spaces at its ends are cut, `ErrInvalidNote`
  (`InvalidArgument`) for none. The EU's rules for online platforms ask for this statement of reasons, as for a
  rejection. `catalog.ErrRemoved` (`FailedPrecondition`) for one removed already. The note is a row of `flags.closures`
  (`flag`, `note`), which a removed flag alone has (`Snapshot.Restore` refuses one without the other): the flag store
  writes it from the flag's snapshot, and deletes it when the flag comes back. It is told to nobody: `FlagClosed`
  carries no note, and planet and the pages need none. **The migration that brought the removals**
  (`20261126153620_flag_removed`, the enum value alone, since postgres uses a new value only once it is committed, then
  `20261126153621_flag_holders`) made each flag closed with a note removed; a flag closed with none (production's
  test Bavaria, closed before the notes existed) stays closed, free to claim.
- **`RenameFlag(code, name)`** (`rename_flag_usecase`): open or closed; `catalog.ErrNameTaken` or `ErrInvalidName`
  (`InvalidArgument`) for a name another flag has or no name. Told as `FlagRenamed`: the pages show it at once, and the
  code, the picture, the land and the players stay.
- **At its end a flag closes by itself**: `expire_flags_usecase`, every 30 seconds, closes each open flag whose
  `ends_at` passed (`Store.Ending`, an index on `ends_at` of the open ones), at its end, and records `FlagClosed`.

## Submissions

**A signed-in player submits a flag, and the operator reviews it** (`internal/flags/internal/submissions`, its own
package and store: a submission has a life of its own, from its submission to its verdict). **Pay first, review after**
(the owner, 2026-11-24: a buyer who waits a day to pay may change their mind): while the shop sells, the founder pays as
it sends its flag, the time waits for the review, an approval opens the flag with it, and a rejection is refunded by
the shop ([Paid time](#paid-time)). **The review rule** (the owner, 2026-11-27): a custom flag opens when it looks like
a flag, is not too political and has a name that reads as a state's, real, historical, fictional or invented
([The judge](#the-judge)). Until that day it was "the flag a real place below a country is known by".

- **Who submits**: an account signed in with Google, Discord or an emailed code (`submissions.FounderOf`, the token's
  `linked`), never a guest, whose account is deleted after 90 days away: a flag needs a founder that lasts.
  `ErrGuest`, `PermissionDenied`.
- **One submission of an account waits at a time** (`submissions.Submit` reads the account's earlier ones, and so does
  an appeal, `Submission.Reopened`): `ErrWaiting`, `AlreadyExists`. Once it is decided the account may submit another.
- **A player changes its submission while it waits, never once the operator decided it** (the owner, 2026-11-24: a
  founder's change of an approved flag "shouldn't be allowed"). The change is a new submission that `replaces` the one
  that waits: in one transaction the old row is deleted (`Store.DeleteWaiting`, which refuses one no longer waiting)
  and the new one kept, then the old picture dropped from the private bucket. It waits at the back of the queue, and
  the operator's approval of the old id is `NotFound`: the operator never approves a picture it did not see. A
  replacement of one decided, or of none, is `ErrNotWaiting` (`FailedPrecondition`); **of one paid for, `ErrPaid`**
  (`FailedPrecondition`): its time is held for it; **of one a person looks at again, `ErrAppealed`**
  (`FailedPrecondition`). Until that day a founder could
  send a new name or picture for a flag it founded, reviewed again; `20261124145305_no_changes` deleted those rows
  (an approved one was on its flag already; a waiting or rejected one's picture stays private in the bucket) and the
  `kind` column.
- **A submission** is a name (`catalog.Name`), a picture and a pin (`catalog.Pin`). A row of `flags.submissions`:
  `id`, `account_id`, `flag` (the flag opened from it, empty until then), the name, the pin, the picture as drawn
  (`picture_*`, with `picture_other_colors`), `picture_held`, `state` (`submission_state`: `waiting`, `approved`,
  `rejected`, `opened`), `submitted_at`, and `decided_at`, when it took its state. A rejection's reason and note are a
  row of `flags.rejections`, which only a rejected one has.
- **Its picture goes through the redraw of the flags sold by hand** (below), at the submission, and is held in private
  under `flags/submissions/<id>.png`, never public before its approval. The operator sees it through a link to the
  private bucket good for an hour (`KeptSubmission.picture`). The upload is deleted once the submission is kept.
- **How many colours it holds** (`pictures.OtherColors`, `KeptSubmission.other_colors_percent`): the share of the picture,
  in percent, in colours other than its 8 most common, each channel in 16 steps. A photo is no flag. Measured on
  2026-11-24: the 240 flags the game ships with (drawn from their SVGs) hold 0 to 10, photos 24 to 65. The backoffice
  warns from 15. Counting the colours that cover one pixel in a thousand did not split them as well (flags up to 35,
  photos from 65).
- **A deleted account's submissions keep its id**, as the ledger's takes do: once the account is purged, the id names
  nobody. Hearing `player.v1.PlayerForgotten` would make the flags module speak player's language, and player already
  speaks the flags' (a cycle).

### The player's calls

On `flags.v1.FlagService`, behind its own session check (`flags_session_checks`): `CreateSubmissionUpload` and
`SubmitFlag` take the click token (`Attested`), `ListMySubmissions` any token that names the account (`Identified`).
`ListFlags` and the stream read no token, as before.

- **`CreateSubmissionUpload(content_type, size_bytes)`**: a one-time link, as the operator's, for a signed-in account
  alone. 20 a day for an account and for its address together (`flags.submissionUploads`, `ResourceExhausted`): each is
  room in the bucket.
- **`SubmitFlag(name, upload_id, pin, replaces)`** (`submit_flag_usecase`): with `replaces`, the id of the account's
  submission that waits, which the new one replaces (above). The refusals of a bad name, pin, picture or id are
  `InvalidArgument`, pictures off or a replacement of one decided `FailedPrecondition`.
- **`AppealSubmission(submission_id)`** (`appeal_submission_usecase`), with the click token: a person looks again at
  the judge's rejection ([Ask a person](#ask-a-person)).
- **`ListMySubmissions`**: the account's submissions, the newest first, whether each waits, was approved, opened or
  rejected, with the reason and the operator's note, `rejected_by_judge` for a rejection the judge made that no appeal
  followed, and `appeal` once its player asked a person. Its picture only from its approval, at its public address. None
  for a guest. Beside them, `held`: the code of each flag the account holds now (`flags.holders`), and `closures`: the
  note of each of those the operator removed. **Only its holder reads it**: `ListFlags` and the stream, which every
  page reads, carry none. **The notice is in the game, never by email**: the page reads it (see the frontend's
  [Flag submissions](../../frontend/docs/flag-submissions.md)).

### The operator's calls

On `flags.v1.AdminService`, the backoffice's inbox and Claude's tools (`flags_list_submissions`,
`flags_approve_submission`, `flags_reject_submission`, `flags_open_submitted_flag`). **Never approve or open a player's
flag in production unless the owner says so.**

- **`ListSubmissions`**: every one waiting, the oldest first, then the 500 decided last, the latest first, each with its
  account, its picture (the private link, or the public one once approved), its share of other colours, the judge's
  judgement and its player's appeal (`KeptSubmission.appeal`, the judge's rejection it asks about). **While the shop
  sells, only the waiting ones paid for or appealed** (`flags.paidReview`, which `main` sets from `shop.Config.Sells`;
  `submissions_query.PaidWaiting`): one whose Stripe page was closed unpaid, or refunded, waits unseen until it is
  paid.
- **`ApproveSubmission(submission_id, pin)`** (`approve_submission_usecase`): with `pin`, the capital where the operator
  moved it on the backoffice's map (`Submission.Moved`, a waiting submission alone; unset keeps the player's, never a
  move to 0, 0). Then the picture held is read, checked against the
  hash the submission keeps (`ErrNotThePicture`), and written public at `flags/pictures/<sha256>.png`, as a flag sold by
  hand; then the approval is kept with `SubmissionApproved` in one transaction, and the private copy deleted. **One paid
  for opens when the approval is heard**, with the time held; another waits to be paid for.
  `ErrUnknownSubmission` (`NotFound`) for one its player replaced, `ErrDecided` (`FailedPrecondition`) for one decided
  already, and `catalog.ErrNameTaken` (`FailedPrecondition`) for a name another flag has: the operator rejects those
  as `exists`.
- **`RejectSubmission(submission_id, rejection)`** (`reject_submission_usecase`): a reason from a short list
  (`not_a_place`, a name that is no state's; `exists`; `unreadable`, not like a flag; `hateful`; `other`, a flag too
  political among others) and a note of 500 characters at most, which `other` needs. The EU's rules for online platforms ask for this statement of reasons. The rejection is kept with
  `SubmissionRejected` in one transaction: **the shop refunds what was paid for it**. **The picture never
  becomes public, and is deleted 30 days later** (`delete_rejected_pictures_usecase`, every hour: the file first, then
  `picture_held` false, so a row that says so holds none).
- **`OpenSubmittedFlag(submission_id, ends_at)`** (`open_submitted_flag_usecase`): the operator opens an approved
  submission by hand, for a flag sold by hand; a founder who pays opens it through the shop ([Paid time](#paid-time)). A name another flag took since the approval is
  `ErrNameTaken`. The flag takes the submission's name,
  picture and pin (`Submission.Open`), and the submission is `opened` with the flag's code, in one transaction with
  `FlagOpened`. `ErrNotApproved` (`FailedPrecondition`) for any other submission.

## Paid time

**The shop tells each payment, and the flags module grants its time once** (`internal/flags/internal/grants`, its own
package and store: a payment's time has a life of its own, from its grant to a refund). The shop is downstream: it
asks and commands in the flags' words, and the flags module hears no module (see [The shop](shop.md)).

- **`flags.v1.InternalService/GetPayable(account_id, target)`** (`get_payable_usecase`), before the shop makes a
  checkout: a submission (`TimeTarget.submission_id`) its founder alone pays for, while it waits or once approved with
  no time paid, or once opened, as its flag (`Submission.PayableBy`; `ErrPaidAlready` for time paid already,
  `ErrNotPayable` for a rejected one); a flag (`TimeTarget.flag`) its holder, or any account once its time ended
  (`Flag.PayableBy`: `ErrHeld`, `PermissionDenied`, for an open flag another account holds or one opened by hand;
  `ErrRemoved`, `FailedPrecondition`, for a removed one). It answers `same_flag`: the target asked and the flag or the
  submission that opened it (`Store.Opened`), so the shop sells no second plan while a renewal of either runs, whoever
  bought it ([The shop](shop.md)). `NotFound` for neither.
- **`flags.v1.GrantTime{grant_id, target, period, paid_at, account_id, ordered_at}`**, a command (`grant_time_handler`,
  group `handler` of `commands:flags.v1.GrantTime`), granted by `grant_time_usecase` in one transaction:
  - a **submission that waits holds the time** (`Grant.ToSubmission`, `flags.time_held`): nothing opens, and the
    submission reads `paid` (`ListMySubmissions`, `Submission.Paid`, read from the grants, never saved with it);
  - an **approved submission** opens its flag (`Submission.Open`, as `OpenSubmittedFlag`), held by its founder, until
    now plus the period, and `FlagOpened` is recorded;
  - an **opened submission**, or a **flag**, is paid for by `account_id` (`Flag.Paid`, [Holders and claims](#holders-and-claims)):
    its holder's ends a period later from its end while it is open, a renewal too, or opens from now once closed;
    another account's claims a closed flag, open from now and held by it; `FlagOpened` is recorded at each opening.
  - The period (`grants.Period`) is a week, or a calendar month or year (`AddDate`), as Stripe bills a plan.
- **Once per payment**: `flags.time_grants` keeps each grant by its id (`stripe:` and the payment's), its flag and how
  much later the flag ended for it (`added`), under an advisory lock on the id. A grant told again grants nothing.
- **`flags.v1.TakeBackTime{grant_id}`** (`take_back_time_usecase`): a refund in full or a dispute lost. The flag ends
  `added` sooner (`Flag.Shortened`), and closes now when that end is past, told as `FlagClosed` (never at the past end,
  which could be before its opening: planet keeps the fact with the latest time). A closed flag stays as it is, and so
  does a flag that opened again since the grant: that time was spent, so an old holder's refund never cuts the time
  of the player who claimed the flag after. Kept in
  `flags.time_takebacks`: told twice, it takes back once; told before its grant (two streams, no order between them),
  the grant comes to nothing.
- **The time held opens the flag at its approval** (`release_held_time_usecase`, on `SubmissionApproved`, group
  `flags-release-held`): each payment held, the first paid first, is granted from now (`Grant.Released`): the first
  opens the flag, the next ones end it a period later, then `FlagOpened`. So the time starts at the approval, never at
  the payment. Told again, the grants released are held no more, and nothing opens twice. A refund before the review
  takes the held time back with nothing to shorten.
- **A payment the rules refuse** (`grants.Refuses`: a submission rejected or gone, a removed flag, an open flag another
  account holds, a flag whose time ended after the order, a name another flag took since the approval) buys nothing
  for good: it is kept as taken back (`Grant.Refused`, so its refund takes nothing back), and `TimeRefused` tells the
  shop, which refunds it in full and cancels the buyer's renewal of it. Logged as a warning, "a payment granted no
  time: the shop refunds it". The shop checked it before the checkout, so only a race or a late renewal leaves one: a
  payment that lands after the judge rejected the flag, its page closed a moment too late; two players claiming one
  flag at once; a renewal Stripe charged late, after another player claimed the flag.
- **The submission's read is locked** (`FOR UPDATE OF s`) in the transaction that changes it: a payment and the
  operator's `OpenSubmittedFlag` never open one flag twice.

`paid_test.go` sends the shop's commands through Redis: the first month opens the approved submission's flag for an
open page, a renewal told twice adds one month, and refunds take each month back, the first closing the flag, which
another account then claims; a month paid while the flag waits opens it at its approval, two days later, for a month
from then.

## Holders and claims

**A flag lives only while someone pays for it** (the owner, 2026-11-26). Until then a founder kept its flag: a closed
Bavaria came back for its founder alone, for 30 days, and its name was taken for good, so no new Bavaria could open.

- **Its holder** (`catalog.Flag.holder`, a row of `flags.holders`: `flag`, `account_id`): the account whose payment
  opened it (a submission's founder) or claimed it last. A flag opened by hand (`OpenFlag`; Brittany, England, Scotland
  and Wales) has none. It stays once the time ended, until another account claims the flag, and on a removed flag, so
  the holder reads its note. The migration (`20261126153621_flag_holders`) gave each flag opened from a submission its
  founder. A deleted account's row keeps its id, as its submissions do.
- **While a flag is open, its holder alone adds time** (`Flag.PayableBy`, `ErrHeld`): paying for an open flag another
  account holds is a later step.
- **Once its time ended, any signed-in account may claim it, forever** (`Flag.Paid`): a payment for the flag (shop
  target `flag`) opens the same flag, its code, name, picture, pin, land and players, with no new review, and makes the
  payer its holder. The old holder may claim it back the same way. The shop refuses a claim while any renewal of the
  flag or of the submission that opened it still runs, a `past_due` one too while Stripe tries the card again.
- **A payment ordered before the flag's time ended claims nothing** (`ErrOrderedBefore`, the order's `ordered_at`
  against the flag's `decided_at`): a late renewal of a plan bought by the old holder, charged after another account
  claimed the flag, is refused, refunded and its renewal cancelled, open or closed again since. The holder's own late
  renewal still opens its flag again.
- **A removed flag is no one's to buy** (`ErrRemoved`), and its name is free.

`flag_test.go` holds each rule; `grant_time_usecase`'s tests a claim by another account, the holder changing, the old
holder's late renewal refused once another claimed the flag, open or closed again, and its refund leaving the claim's
time; `get_payable_usecase`'s who may pay; the shop's `create_checkout_usecase` a claim refused while a renewal runs.

## The judge

**A Claude model reviews each flag as it is sent, while its buyer is on Stripe's page** (the owner, 2026-11-24:
"by the time Claude provides its answer the guy finished the checkout"), `internal/flags/internal/judgements`, its own
package and store. Off with no key (`flags.judge.anthropic.apiKey`): the operator reviews every flag, as before.

- **Every 3 seconds** (`judge_submissions_usecase`, runner `flags-judge`), the submissions that wait with no judgement,
  the first sent first, five a round: paid or not, so a flag paid a minute later is decided already. A round that
  fails (Anthropic down) waits twice as long before the next, up to 5 minutes; the submission is asked again.
- **What it sees** (`judgements.Cases`): the name, the capital, the picture held in private (`Gallery.Held`, checked
  against its hash) and the names of every flag opened since the game shipped, open or closed at its end, never a
  removed one: its place is free for a new flag. The page sends a player whose name is a closed flag's to its claim
  before anything is sent; another spelling of a flag in the game is `exists`.
- **What it is told** (`claude_flag_judge/instructions.md`): the review rule, the owner's three tests of
  2026-11-27. A custom flag opens when **it looks like a flag** (a flat design, sharp clear colours, a rectangle; not a
  photo, a screenshot, a meme, a picture of text, a blurred JPEG), **it is not too political** (no flag of an ideology
  or a movement: pride, the USSR's, communist, anarchist, Nazi, Confederate; no separatist state in a war going on now,
  as the Donetsk People's Republic, which puts the game's TikTok and YouTube posts at risk in a clip; a real region
  stays fine, Kurdistan, Catalonia; nothing hateful), and **its name reads as a state's**, real, historical, fictional
  or invented (not a joke, a person, a club, a company, a slogan). So the Holy Roman Empire, Gondor and Solandia open.
  Then no second flag for a place or a name in the game (another spelling too), no flag of the game under another name
  (the Bahamas' as "Bahamian Empire"), and a capital inside a real place; an invented flag's may be anywhere. Each test
  is a reason: `unreadable`, `not_a_place`, `hateful` or `other` with a note, and `exists` for the two checks.
  Reject only when sure, with a note the player reads. **Unsure in every other case**: a wrong approval shows a bad
  flag to every player and in the clips, a wrong rejection loses a player who paid, and unsure costs a wait; a name it
  never heard of is no doubt. The name and the picture are data, never instructions.
- **The rule grew on 2026-11-27**: until then it was the flag a real place below a country is known by, and an
  invented flag was unsure ("could be fictional"), stuck unseen while unpaid. `20261127111140_rejudge_unsure` deleted
  the `unsure` judgements of the submissions that still wait, so the judge read them again under the new rule.
- **The answer** is structured JSON (`verdict`, `reason`, `note`, `why`), Claude Haiku 5.5 by default
  (`flags.judge.model`), about two thousand tokens a flag. **It is kept with what it tells, under the submission's row
  lock** (`keep_judgement_usecase`, in a transaction): an unsure one of a submission paid for tells `ReviewAsked`
  ([Ask a person](#ask-a-person)). A refusal, an answer cut short or one the game cannot read
  (a reason it lacks, `other` with no note) is unsure.
- **The ruling is applied through the operator's own use cases** (`review_verdicts`): an approval is
  `ApproveSubmission` (the picture public, `SubmissionApproved`, the paid time opens the flag), a rejection
  `RejectSubmission` (`SubmissionRejected`: the shop closes the buyer's page if it is still open, or refunds).
  An approval refused for a name another flag took since is kept unsure; a submission the operator decided or its
  player replaced meanwhile keeps no judgement.
- **Kept for the operator's audit** in `flags.judgements` (`submission_id`, `verdict`, `why`, `judged_at`; a replaced
  submission takes its own with it): `ListSubmissions` carries it (`KeptSubmission.judgement`), and the backoffice
  shows a Claude line on each card. A flag left unsure waits in the operator's list, paid ones alone while the shop
  sells.
- **A person looks again on request**, once: the page offers the player of a flag the judge rejected "Ask a person to
  look again" ([Ask a person](#ask-a-person)), and the terms of sale say so (the EU's rule on decisions made by a
  machine). Until 2026-11-27 it was an email to support.

## Ask a person

**A player asks a person to look again at the judge's rejection, once** (the owner, 2026-11-27: no player stuck on a
refusal), `internal/flags/internal/appeals`, its own package and store: an appeal has a life of its own, asked, then
answered by the operator's verdict.

- **`appeals.Ask`** (`appeal_submission_usecase`, `FlagService/AppealSubmission`, the click token): the founder alone
  (`ErrNotYours`, `PermissionDenied`, a guest `ErrGuest`), of a rejection the judge made (`Judgement.Rejected`; one the
  operator made, or one the judge was unsure of, is `ErrNotTheJudges`), once per submission (`ErrAskedAlready`), while
  no other of the account's submissions waits (`ErrWaiting`, `AlreadyExists`) and its picture is held
  (`ErrPictureGone`: it is deleted 30 days after the rejection); `FailedPrecondition` for the others.
  `Submission.Reopened` puts it back to waiting, its rejection's row dropped, its `decided_at` the ask's; the appeal
  keeps the rejection it asks about. All in one transaction with `ReviewAsked{cause: APPEAL}`.
- **Never judged again**: its judgement stays (`reject`), and the judge reads only the submissions with none.
- **The operator sees it even unpaid** while the shop sells (`ListSubmissions`, `appeal`), with the judge's reject
  line. Its approval asks the player to pay to open it, as an approval not paid for; its rejection is final:
  `rejected_by_judge` is false once an appeal was asked, and a second appeal is `ErrAskedAlready`. No money moves at
  the appeal: the payment was refunded at the judge's rejection.
- **Answered by the verdict** (`answer_appeal_usecase`, on the module's own `SubmissionApproved` and
  `SubmissionRejected`, groups `flags-appeals-approved` and `flags-appeals-rejected`): `asked` becomes `answered` at the
  verdict's time. A verdict from before the ask (the judge's rejection, heard late) or told again changes nothing.
- **Not changed while a person looks** (`ErrAppealed`): a new flag does not replace it.
- **`flags.appeals`**: `submission_id` (one per submission, deleted with it), `account_id`, the judge's `reason` and
  `note`, `state` (`appeal_state`: `asked`, `answered`), `asked_at`, `decided_at` (`20261127112509_appeals`).

**`ReviewAsked{submission_id, name, cause, why, asked_at}` tells that a flag waits on a person**, once for each cause,
which the alerts module posts to the staff's private Discord channel (see [Discord pings](alerts.md#the-staffs-channel)):

- `REVIEW_CAUSE_APPEAL`: its player asked a person, from the appeal's transaction.
- `REVIEW_CAUSE_UNSURE`: the judge was unsure and its time is paid (`Judgement.Facts`), whichever came last: the
  judgement (`keep_judgement_usecase`) or the payment held for a waiting submission (`grant_time_usecase` reads the
  judgement). Both read the submission `FOR UPDATE`, so exactly one of the two sees both halves. An unsure flag never
  paid for is told nothing: it waits unseen while the shop sells.
- `why` is the judge's, in its words.

`appeal_test.go`, the use cases' tests, `submissions_query`'s test of a flag appealed shown unpaid, and the stores'
contract suites hold it.

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
- **A submission's picture is held in private** (`Gallery.Hold`, `Cache-Control: private, no-store`) under
  `flags/submissions/`, and made public by `Gallery.Release` at its approval. A file uploaded and never submitted stays
  under `flags/uploads/`: a lifecycle rule of the bucket on that prefix would delete it (an operator step, not done).

`internal/flags/told_test.go` opens a flag through the module with a real postgres and Redis, and shows the fact reach
another module's subscriber and an open page's stream with no restart, then the close and a rename. `submitted_test.go`
takes a player's flag from its submission, changed while it waits, to an open page, the old one's approval refused,
then refuses a change of the flag opened.

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
  at boot tells it to planet. Its pin is Rennes' (48.1173, -1.6778), which the next migration
  (`20261124101101_flag_submissions`) gives it by name.
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
3. Then the owner's call: keep it until a date (`flags_add_time`, `xb`, the days past its 30), or remove it now
   (`flags_close_flag`, `xb`); `flags_add_time` brings it back.

## England, Scotland and Wales, from the build to the sold flags

**England (`gb-eng`), Scotland (`gb-sct`) and Wales (`gb-wls`) shipped with the game until 2026-11-24.** The owner
decided that day that they are sold like Brittany. One deploy moves them, as it moved Brittany, and each keeps its code,
so nothing of them is rewritten: land and shields, pledges, chat rooms, season points, ledger rows. In production that
morning none held land, nobody had taken a tile for them in 72 hours, and none had a flag day in the week; Wales had 1
point of Season 1, England and Scotland none.

- **The build no longer lists them**, as Brittany: `countries.json` (`make flags`), the atlas, the SVGs, the fit, the
  names in the page's languages, the backoffice's own names and its emoji of the three nations. With them went the
  last built-in flags with no ground: `cpflags.Pins` and its `make flag-pins`, and the map audit's `REGIONS`.
- **The flags module holds them from the first boot of that build** (`20261124143335_regions_sold`): open, named
  "England", "Scotland" and "Wales", ending 30 days after the deploy (the owner's choice), each picture the build's SVG
  rendered at 1500 x 900 with `rsvg-convert` and drawn by `pictures.Redraw`, 768 x 461, each pin its capital as
  `cmd/flagpins` read it from Natural Earth's populated places: London (51.501941, -0.118668), Edinburgh (55.948328,
  -3.219091), Cardiff (51.499995, -3.225008). The restatement at boot tells them to planet, pins and all.
- **Planet hears them open, on their pins, in the same deploy** (`20261124143336_regions_heard`), so their players are
  never told their flag closed, and a Capital Siege opened before the restatement arrives still seats them.
- **Their pictures go into the bucket by hand, once**: until then the page draws them as the blank.
- **What they lost**: their Discord roles (the alerts' tidy at boot deletes them), their names in other languages, their
  old addresses (`clickplanet.lol/gb-eng/<name>`). Their 85 quiz questions (31 England, 33 Scotland, 21 Wales) moved to
  subject `gb`, their ids kept, so their ratings stay. England keeps the United Kingdom's anthem: `anthemsAsset.ts` maps
  a code, whatever the build lists.

Production, in order, each with the owner's yes:

1. Put each picture in the bucket `clickplanet-media`, `Content-Type: image/png`, `Cache-Control: public,
   max-age=31536000, immutable`, at `flags/pictures/<sha256>.png`: England
   `a5bbe1742db6aa08e5850c063ca9feb11b17c99fbf2546f07ed1f2e57bb44edb`, Scotland
   `e044e22d0d52a24b10503b26b85c66d0c9223e2ddd07842b65d0c9b00ab0a5f8`, Wales
   `625234995a5a9c1da4e98805b1f62053d49ee570cc82c79494ae5e6914fb32d5`. The files are on the PR's assets branch.
2. Deploy: frontend first, then the backend.
3. Then the owner's call for each: keep it longer (`flags_add_time`), or remove it (`flags_close_flag`);
   `flags_add_time` brings it back. Since 2026-11-26, once its time ends, any player may claim it.

## The pool

The flags module holds 1 postgres connection; player went from 4 to 3, so `max_connections` stays 42.
