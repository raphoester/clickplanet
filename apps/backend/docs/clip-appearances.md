# The players a clip names (`internal/clips`, the server's side)

**A clip ends on its top fighters, and the game tells each of them** (the frontend's
[Clips](../../frontend/docs/clips.md) and [In a clip](../../frontend/docs/clip-appearances.md)). A player who sees
their name in a clip shares it. The clips module of the server keeps what this needs, in its `clips` schema, beside
the [Clip stats](clip-stats.md): **the cast of each run, and who each post names.**

**An account id never leaves the server.** The run's files are public artifacts of a public repo (the backend's
[The clips job](clips-job.md)), so the job is handed numbers and names, and gives numbers back.

- **`clips.v1.AdminService/MakeCast(since, until)` makes a run's cast** (`casts/`, `make_cast_usecase`), on the
  console listener, which `clips fetch -cast` asks over ssh for the replay's window (72 hours at most: the ledger
  keeps who made a change that long). It reads who took what from planet (`planet.v1.InternalService/GetLedger`, the
  admin's handler and all, page after page, through `rpc_planet_ledger`; `LedgerChange.account_id` is the account of
  each change), and the name each account may be shown under from player (`InternalService/GetAuthors`, through
  `rpc_player_names`: an `Author` with `shown_in_clips`, a username its player did not keep out of the clips; see
  [Player](player.md)). **A fighter is a player with such a name who took a tile from another flag**: filling empty
  ground is no fight (`Take.captured`). The fighters are numbered from 1 by tiles taken, then by name
  (`casts.CastOf`). The answer is the `clips.v1.Cast`: its id, its window, each fighter's number and name, and each of
  their takes from another flag under its number. A guest, a player who turned the clips off, and a change of no
  account are in no cast.
- **The cast is kept 3 days** (`casts`, `cast_fighters`): which account each number is, for the posts of the run to
  name. A run posts its clips within hours, so making a cast forgets every cast made more than 3 days before
  (`casts.ForgottenBefore`, then `Forget` by id: the domain decides, the store finds and deletes). The id is a random
  uuid, not a v7: it is in the public files, and says nothing of when it was made.
- **`KeepAppearance(cast_id, fighters, post_id, title, platforms, scheduled_at)` keeps who a post names**
  (`appearances/`, `keep_appearance_usecase`), which `clips schedule` asks over ssh once a post is scheduled, with the
  platforms that took it. The numbers are read back through the cast, one row a player (`appearances`), kept again as
  one. **`NotFound` for a cast forgotten or never made, `InvalidArgument` for a number it does not have**, and the
  job logs a warning: the clip is out already. Keeping one forgets the appearances scheduled more than 7 days before
  (`appearances.ShownSince`).
- **`clips.v1.ClipsService/GetAppearances` is the public read**, the one service of the module on the public router:
  the posts out in the last 7 days that name the caller, the newest first, each with its headline, its time and a
  link per platform (`appearances_query`). It needs the identity token (`clipsv1controller.NewSessionInterceptor`,
  `Identified`): a player reads its own, and no request names an account. **A post scheduled for later is not out**,
  so the game never tells a clip before it plays. **A link is the video's address once a snapshot of the stats read
  it** (the newest `snapshot_rows.document->>'url'` of that post and platform, `snapshot_rows_post`), else empty: the
  daily snapshot is the one read of Zernio that has it, and the game links to the channel until then.
- **A deleted account is forgotten** (`auth.v1.AccountDeleted`, so the module subscribes to Redis): its appearances
  are deleted (`clips-appearances-accounts`), and in a cast its number stays and names no one (`clips-casts-accounts`,
  `account` and `name` set to NULL), so a post of that run keeps the others.
- **What it does not undo**: a player who turns the clips off after a run's cast is still named by that run's clips,
  posted within the next 3 hours, and a posted clip stays on TikTok and YouTube.
- **Tests**: the domain and the use cases on in-memory stores (`casts.MemoryStore`, `appearances.MemoryStore`, behind
  the `testing` tag), the adapters on fakes of planet and player, both stores and the query on postgres
  (`postgres_appearance_store`'s suite reads what it keeps through `appearances_query`), and the module booted with
  postgres and Redis (`module_test.go`).
