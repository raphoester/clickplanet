# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

A Go backend for a collaborative map-clicking game, in **hexagonal architecture (ports & adapters)**. This file
is only an index (see the root [Docs rule](../../CLAUDE.md#docs-rule)): each line names a doc in [`docs/`](docs)
and says what is in it. Read the doc before you change its area.

## Working on it

- [Commands](docs/commands.md) — `make test` (Docker for the postgres and Redis tests), `-race`, `make deadcode`,
  `make lint`, `make tidy`, `make check-format`, `go run ./cmd/api`, `make proto`, the vendoring targets (`quiz`, `flags`,
  `map`, `moderation`, `vpn-lists`, `disposable-domains`, `historic-scripts`, `zernio`), `make dBuild`,
  `cmd/clips`, `cmd/teasers` and `cmd/translations`.
- [Configuration](docs/configuration.md) — `cpconfigs` loads YAML and environment with koanf; the full schema.
- [Testing](docs/testing.md) — testify; which suites need Docker (postgres, S3, Redis); `e2e/` holds smoke tests
  only, rules go in lower tests; the `testing` tag.
- [Linting](docs/linting.md) — golangci-lint at a pinned version, gofumpt, `.golangci.yaml`, and the linters left
  off on purpose.
- [Git hooks](docs/git-hooks.md) — what the repo's hooks run for this app.
- [Protobuf](docs/protobuf.md) — the contracts in the shared `/proto`, generated into `generated/proto`.
- [Migrations](docs/migrations.md) — a module's golang-migrate files and their UTC-time versions; what golang-migrate
  skips or refuses at boot; `make check-migrations` and the deploy's check against the revision the VPS runs.

## How the code is shaped

- [Nineteen bounded contexts, one process](docs/bounded-contexts.md) — flags, planet, boxes, bonuses, quizzes, chat, auth, player,
  progression, scoring, support, missions, antibot, gamemaster, alerts, clips, promo, publishing and vandalsquad. A module publishes its root package and hides the rest; the
  modules depend on each other with no cycle, and only player and alerts speak auth's language.
- [The composite layer](docs/composite-layer.md) — each context wires itself in its `module.go`; how a module
  calls another one, how it tells the others what happened and asks one to change something: Redis Streams, an event
  moved to another package (`SubscribeMoved`), the idempotent receivers, the outboxes, the commands, and the
  gamemaster's calls that answer at once.
- [Inside the planet module](docs/planet-module.md) — one directory per concept, no `domain` or `adapters` layer;
  use cases.
- [Inside the chat module](docs/chat-module.md) — the same shape, for chat's ten concepts.
- [Reads are queries](docs/queries.md) — a read endpoint does not go through the write model.
- [Adapters](docs/adapters.md) — the primary (Connect handlers) and secondary (stores) adapters.
- [Transactions](docs/transactions.md) — a use case opens one with `InTx` and records its events in it; the
  smallest one possible, one change and its event, and every other change eventually consistent through events unless
  it must be atomic; a store reaches postgres only through `db.On(ctx)`.
- [Every id is a type of its own](docs/ids.md) — never a bare `uuid.UUID`, `string` or `[]byte` in a signature.
- [Manipulators are verbs, builders are nouns](docs/verbs-and-nouns.md) — a name says if calling it changes
  anything.
- [A command's `run` is a table of contents](docs/command-run.md) — a CLI `main` is named stages, and `run` only
  wires them.
- [Shared](docs/shared.md) — the `internal/shared/` `cp*` infrastructure packages.
- [The error net](docs/error-net.md) — no handler's raw error reaches the wire.
- [Shared interceptors](docs/interceptors.md) — the `shared/cpconnect` interceptors that all contexts use.
- [A server restart reloads the page](../../docs/reload-on-restart.md) — `X-Server-Boot` on every public answer, drawn
  at each start; a wire change is a hard switch.

## The tile game (planet)

- [Key flow](docs/key-flow.md) — the path of one click, step by step, from the mint to the stream.
- [The map load](docs/map-load.md) — `GetMap` is a cached HTTP GET; the stream carries what follows.
- [The map reset](docs/map-reset.md) — a season after season 0 starts on a blank map: the gamemaster's `ResetMap` call,
  wiped once per reset on a context the caller cannot cut, a new epoch, and replays that start at the reset.
- [Arena battles](docs/arenas.md) — a battle whose set has `arena.on` plays on a fresh map, and the map from
  before comes back at its end; the arena is a shell: it switches the map and seeds it (`seeds`, the capitals pattern);
  the order that waits on answers kept (day K scored, the arena opened, the battle's set played after, the map back),
  the `OpenArena` and `CloseArena` calls whose swap the caller cannot cut, the
  freeze, the gaps; the ledger's readers (replays, map history, clips, the holder card) skip it by the map each event
  names, so the world's map before and after joins up.
- [The reach](docs/reach.md) — a knob: a take only next to the flag's land or across a bridge from it, on any map; why
  bridges and not the waters.
- [The capitals' rules](docs/capitals.md) — three knobs: no shield on a capital, the knock-out (a capital taken clears
  its flag's land in one event, the flags out kept with the tiles, the pick of a flag still in), and the cut-off (land
  a take cuts from its capital, over the flag's tiles and bridges, cleared in one event); the seats an arena gives the
  sold flags from their pins, played as capitals.
- [The map's version](docs/map-version.md) — every write counts one, published under the tiles lock so streams get
  them in order; `GetHoldings` and the one-second `holdings_changed`; the ETag and the 304.
- [The live streams](docs/live-streams.md) — the `ListenForEvents` streams, served two ways, and how they end on
  shutdown.
- [Durability](docs/durability.md) — the map lives in memory and goes to postgres every second; what a crash
  loses.
- [Map geography](docs/map-geography.md) — a board's ground: positions rebuilt from the runs and checked against the
  manifest's hash, adjacency taken from the lattice, each file's tile count checked as the board loads.
- [Boards and eras](docs/boards-and-eras.md) — the registry of every board carried, each loaded when first needed;
  `planet.boards` and `eras`, the `board` column on every row of a tile and the `map` and `board` of every event
  (`clicks.Stage`), the backfill and its test, the gate that keeps a write and its record on one stage, the ground in
  play swapped with the stage (`clicks.Played`), the boot's guard (every board carried by name and hash, rows of
  another board refused), each reader by column, the history drawing each era on its board with the cut, the board
  on the wire, and the world's switch of board (`SwitchBoard`: the carry-over, one transaction under the locks, the
  `board_switched` event that is no take, `BoardSwitched`, the dry run).
- [Sized from the board](docs/board-scale.md) — `cpboards`' scale against `earth-1`: counts of tiles with the density,
  lengths with the spacing, zooms the other way; the mission floors, the fortify news, the vandal squad's largest
  drawing and link, the attack ping's zoom; unchanged on `earth-1`.
- [Rate limiting](docs/rate-limiting.md) — two buckets per click, what the answer says is left, and a big country
  refills slower.
- [VPN blocklist](docs/vpn-blocklist.md) — `Click` refused from vendored VPN and Cloudflare WARP ranges, before the
  rate limiter; what it costs iCloud Private Relay; `cmd/prefixdiff`.
- [Allegiance](docs/allegiance.md) — an account plays for one flag, and may change it once an hour; a battle's rule
  set locks it; a change the knock-out forces starts no wait, so after the battle the player changes back once.
- [Rules](docs/rules.md) — planet's game rules, in rule sets kept in postgres (the wallet's `charges.*` among them; the
  offers keep their own); `default` holds every value, another set only what it changes; the set in force in one atomic
  pointer; what a change does to what is already there; a frozen set.
- [Shields](docs/shields.md) — a charge on a tile; each shield takes one foreign click.
- [A claim is one lock](docs/tile-claim.md) — `clicks.Tile` holds owner, shields and holder; a claim reads, decides and
  writes it under one lock; why a lock and not a version per tile.
- [A shield's way from the hand to the tile](docs/placements.md) — the placement row between the spend and the map's
  flush; a refusal gives the shield back with no cap; the boot gives back what a stop left.
- [Fortify](docs/fortify.md) — a take that makes a province whole gives every tile of it a shield; the lock per
  province, the switch from landmasses (2026-11-18), the wire, the ledger's old and new `fortify`.
- [Who holds the tiles](docs/tile-holders.md) — each tile keeps the account whose take holds it; `GetLandmass`
  reads a landmass's holders, `GetProvince` a province's last fortify.
- [The map's last 24 hours](docs/recent-map-history.md) — `GetRecentMapHistory`, public: a frame each half hour, rewound
  from the map in memory over the ledger's rows from its head and the erasure kept of the rows before it, built once per
  step; what cannot be rewound; its size; why it names no account.
- [A revert erases the takes it forgot](docs/revert-erasure.md) — every history and replay reads the ledger as if a
  reverted player never played (`ledger.Erasure`): a take of its paint takes the tile from what was under it, a take back
  to it is none; the revert gives back every take of the season on the live map too, on every board (`ledger.Undoing`
  over `world_replay`), and the shields the caller struck; why the history and the live map differed before; the
  erasure before the ledger's head that both rewinds start from (`world_replay.Erasures`); `GetLedger` leaves them out;
  each cache built again after a revert (`forgetting`, `GetMapHistory`'s `kept`); production's rows put right; what is
  left.
- [Charges](docs/charges.md) — every bonus is a charge, kept until the player uses it; every kind a count, stacked to
  its `charges.*.held` (refills, bombs and viruses too since 2026-11-16), none kept of a kind the rules turn off.
- [Bonus boxes](docs/bonus-boxes.md) — the box each caller is offered on a schedule of its own (the boxes module's),
  the kinds it grants (drawn by the bonuses module at the catch), `ClaimBox`, `box_offered` and the hourly cap.
- [Spread](docs/spread.md), [Bomb](docs/bomb.md), [Enclose](docs/enclose.md), [Refill](docs/refill.md) — what
  each charge does to a click or to the bucket.
- [Virus](docs/virus.md) — a charge that takes an enemy tile and spreads every 20 s into that flag's land, 32 infections
  at most; dormant (weighed 0); map state flushed with the tiles; one `virus_spread` a tick; ledger kind `virus`, whose
  takes reach `TilesTaken`.
- [Mine](docs/mine.md) — a charge laid on a flag's own tile that only its players see; any enemy take sets it off and
  its 2-ring blast takes the tiles for the flag, every mine reached going off too; a bomb clears it; dormant (weighed 0),
  level 13; map state flushed with the tiles; one `mines_exploded`, `mine_laid` to the flag's streams alone; ledger kind
  `mine`, whose takes reach `TilesTaken`.
- [Trap](docs/trap.md) — the mine's sibling: a charge laid on a flag's own tile that only its players see; any enemy
  take springs it and the taker loses `charges.trap.clicks` (10) from its bank, to nobody, down to minus one bank
  (`Limiter.Drain`), a guest's network's guest bank too; the click's answer shows the debt; a bomb clears it; dormant
  (weighed 0), level 15; map state flushed with the tiles; `trap_laid` to the flag's streams, `trap_sprung` to every
  stream; no ledger kind.
- [Rally](docs/rally.md) — a charge that calls a ring on any point of the globe, seen by every player: for 3 minutes a
  click of its flag inside costs half a click (`charges.rally.cost`, from every bucket, `ClickBudget.spent`); one on at a
  time, a stack of 3; dormant (weighed 0), level 17; a row in postgres, not map state, that a remap keeps;
  `rally_called` on every stream, `GetRallies`; chat's `rally` line in the flag's room, naming the landmass; no ledger
  kind: the takes inside are plain clicks.
- [Quizzes](docs/quizzes.md) — the quiz banner, its short clock, the bank's lean on planet's board, and the charge a
  right answer gives (the bonuses module's, from the quizzes' `QuizAnswered`).
- [The boxes module](docs/boxes-module.md) — the boxes left bonuses (2026-11-15): the schedule, the offer and the claim,
  a newcomer's early box, the box facts; `BoxService` (pays nothing, no kind on the wire), `BoxClaimed` through an outbox,
  its rule sets (`boxes.rule_sets`), no hand's room asked (a full hand is offered boxes that pay nothing), the two
  changes for a player, the deploy.
- [The quizzes module](docs/quizzes-module.md) — the quizzes left bonuses (2026-11-13): the schedule, the bank, the
  ratings and the answer log; `QuizService` (pays nothing), `QuizAnswered` through an outbox, its rule sets
  (`quizzes.rule_sets`), the tables moved from schema `bonuses`, the dropped "box and quiz apart" rule, the deploy.
- [How hard each quiz question is](docs/quiz-difficulty.md) — a rating per question, moved by every answer counted
  (`P = 1/3 + 2/3·σ(s − d)`, a running mean then a 1/200 step), from the bank's prior; the answer log and the totals in
  postgres, the copy in memory the draw reads; what counts (a late answer too), and what comes later.
- [A newcomer's first box and first quiz](docs/welcome-bonus.md) — a new account gets both early, once (the boxes' and
  the quizzes' timing; a spread or shields, the bonuses module's); the dry click the quizzes hear from planet's
  `ClickTried`.
- [A guest's first sign-in earns a refill](docs/sign-in-refill.md) — once per account, when player makes it a player
  (`PlayerCreated`, which the offers hear and pay through planet's `GrantCharges`); why nobody can farm it.
- [Offers and the wallet](docs/offers-and-wallet.md) — the seam cut inside planet (step A) and the move of the offers
  out of it (step B, done 2026-11-11), then the quizzes' and the boxes' cuts: the wallet (`charges`) and the offers
  (`bonuses`), the ports, what `Clicked` hears, the rules split.
- [Bonuses](docs/bonuses.md) — the rewards' module, downstream of planet like missions: a box caught paid from the boxes'
  `BoxClaimed` and a right answer from the quizzes' `QuizAnswered`, each drawn at the win, and a level-up from
  progression's `PayLevelUp`, which fills every stack (`FillCharges`); the gates (the kinds open to all and the ones
  progression unlocked, `KeepUnlocks`); the sign-in refill; no stream since 2026-11-15; its rule sets
  (`bonuses.rule_sets`, played by the gamemaster), what it asks planet in planet's words (`GrantCharges`: the draw
  ignores the hand since 2026-11-16, and the wallet keeps what fits) and where its sets came from.

## Who a caller is

- [Sessions](docs/sessions.md) — the click token that makes clicking cost something; the click token and the
  identity token.
- [Auth](docs/auth.md) — Turnstile, guest and signed-in accounts (Google, Discord, emailed code), operators,
  sign-out, deletion with its 30-day grace period and the purge, a sign-in that brings an account back, and pruning.
- [One mailbox, one player](docs/mailboxes.md) — `accounts.MailboxOf`: the forms of one address (`+tag`, Gmail's dots,
  googlemail) are one identity, for the providers it lists; the email code's subject, `identities.mailbox` and the owner
  it finds, the budget per mailbox; the migration that folded what was kept and the accounts the forms made.
- [Arrivals](docs/arrivals.md) — how each new account came: a player's link or video, TikTok, YouTube, or
  unknown, from a tracked address's one-letter first segment (`/t`, `/y`, `/r`, `/v`),
  the link's tag (a streamer's own link) and the referrer's host; the recruiter's account, in a table of its own
  (`auth.recruitments`); kept by `auth` when it makes the account, anonymized when the recruit's or the recruiter's
  account goes; the queries.
- [Anti-bot](docs/antibot.md) — a module that hears planet's and the boxes' facts about callers, judges them, and asks planet to
  refuse the ones it bans; a ring of accounts warned once before a ban for its turns, by the jury or by hand; two
  accounts that play in turns from one connection each shown a notice once, which bans nothing; a streamer's chat, which
  `default` keeps from bans with `cohort` and `churner` off; what survives a restart, and the parts that are easy to get wrong.
- [The antibot's rules](docs/antibot-rules.md) — every bound is a rule set in postgres, edited from the backoffice, that
  judges with default's where it holds none; the pointer the watchdogs read, which set plays, what stays in the YAML.

## The other contexts

- [The flags module](docs/flags.md) — the catalog: the flags the game ships with (the page's file, `make flags`) and the
  ones opened since (`r-` ids, no two with one name, closed never deleted); `FlagOpened` and `FlagClosed` through its outbox and again at
  each boot; the catalog planet alone keeps from them (playable or known), the others checking a code's shape
  (`cpflags`); the operator's calls (open with a picture and an end, add days, close, rename), the close at the end date and
  the comeback within 30 days; an early close's note, which the founder alone reads; a closed flag's players changing at once, its land kept, its room read-only, a battle's
  lock keeping it playable; the pictures drawn again, public on media.clickplanet.lol; its own stream to the page;
  Brittany, moved from the build to the sold flags under its code `xb`, and the production steps of that move; each
  sold flag's pin (a point, never a tile), told to planet (`FlagOpened.pin`, `FlagMoved`); a signed-in player's submission (one waiting at a time, its picture held in
  private), the operator's approval (public, a change on its flag at once) or rejection with a reason (the picture
  deleted 30 days later), and the opening of an approved one by hand.
- [Chat](docs/chat.md) — messages, the banned phrases refused at send however they are spelled, each flag's room
  beside the world chat, GIFs, announcements, mutes, removals, the moderation history (every mute, unmute and removal
  with its reason and who made it, kept 90 days), raid mode (no guest posts), the Claude Haiku judge of reports (the
  steps applied in code, one judgement per report and per message, an hourly cap), replies, reactions, and what each
  account saw; why each context keeps its own history.
- [Player](docs/player.md) — who an account is: names (a slur or a Nazi reference refused however it is spelled) and
  guest codes, the names a player left (the last 5 on its card, since 2026-11-17), colors, the nameplates the page draws every name
  from (`GetNameplates`), the player that holds a name (`GetPlayerByName`, for a player's address), who is playing now, and an account
  asked deleted, hidden from the other players until it is purged or comes back; what it tells the other modules of
  an account (`PlayerCreated`, `PlayerNamed`, `PlayerSignedIn`, `PlayerLeaving`, `PlayerReturned`, `PlayerForgotten`,
  `GetAccounts`), so no game module but alerts hears or asks auth; the deploy that moved them to it.
- [Progression](docs/progression.md) — how far an account went: stats, streaks (a missed day the operator forgives),
  titles and their tracks, the title worn, the fronts, the badges, the title-earned stream and the reconciliation; why it left `player`, and its tables
  moved from schema `player`.
- [Levels](docs/levels.md) — a level per linked account, from XP computed from the stats row and never stored: one type
  per XP source and per mission kind's rate, the curve 14 × (L − 1)³ with no top, no XP and no level for a guest; the
  level kept, raised on each fact and told to the caller's own stream with the XP each fact gave by source (`xp_gained`,
  a difference from what the stream knew), the ladder of the kinds each level unlocks, told
  to the bonuses as a command through progression's outbox, the public reads the page draws (`GetLevel` with each
  mission kind's exact rate, what each other source pays and the ladder, the card's level), the level-up bonus,
  `ReconcileLevels` (raise-only), the
  operator's read, what ships together at the reveal, the steps after the deploy.
- [Season titles](docs/season-titles.md) — Season 0 Gold, Silver, Bronze, Battle Top 1-3 and Veteran: who fought the
  Final Battle, where its flag placed and who took the most tiles there, read from scoring when the gamemaster tells
  `SeasonEnded`; Season 0 Top 1-3, its best players, given once by hand.
- [Scoring](docs/scoring.md) — a points accountant: books, rounds and grants by the ids it is given, opened and closed
  by the gamemaster's calls (a round or book never opened answered missed, and kept so no open lands after), each round placed by its ranking: a battle on the countries that fought, by the tiles held and the tiles taken, a knock-out battle on the flags still standing first, then the last out first, a day on that rank plus its rank on the tiles held; no season, day, battle or clock; who a
  board may rank, kept from player's events.
- [The gamemaster](docs/gamemaster.md) — every moment of the game: the rule schedule that calls planet, the antibot,
  the boxes, the bonuses and the quizzes, the seasons, days and battles it opens and closes in scoring's books, the map swaps on a runner of their own, each season's finale (the Final
  Battle), the events in the game's words (each season, day and battle as it starts), and how to freeze the map over
  the break between two seasons.
- [The gamemaster as a reconciler](docs/gamemaster-reconciler.md) — each second, what the game wants against what the
  modules answered: each change an RPC called until it answers, kept right after the call, no call before the ones ahead
  of it answered, a refusal for good, a call cut by its timeout, the reset after the book before it, the boot, the cost.
- [Missions](docs/missions.md) — four or five missions a day per player, in the days the gamemaster tells
  (`DayStarted`), counted from planet's, the boxes' and the quizzes' facts (takes, boxes, right answers, bombs, shields, fortifies), paid once done
  in XP, which progression counts, and no charge (the reveal set); one flag mission a day per flag, worth season points (`GrantPoints` to
  scoring), the fewer the more the flag won in the season (scoring's `GetTable`, the draw waiting for the rounds before
  the day to be scored), and told done to the chat (`FlagMissionDone`), which announces it in the world and the flag's room; why
  progress is not read from the ledger; the flag mission's numbers, one document the operator edits (`GetRules`,
  `SaveRules`); the operator's reads of what was given and done (`ListFlagDays`, `GetMissionTotals`); `MissionsDone`,
  every account's missions done by kind, a guest's too, with no reward in it, and `RetellMissionsDone`.
- [Promo](docs/promo.md) — a battle's teaser: each rule set's cards, kept under its name, over backgrounds the operator
  uploads to the bucket and Cloudflare serves publicly; words with `{path}` placeholders filled from the values the set
  plays and the battle's points at each read (`GetTeaser`); the set and times from the gamemaster (`GetBattle`), the
  values from planet, the boxes, the bonuses and the quizzes (`GetRuleSet`), the points from scoring (`GetRankPoints`); the battles to come with their cards
  (`ListTeasers`, from the gamemaster's `GetTimeline`); each battle's teaser posted on TikTok and YouTube through
  publishing (`ListDueTeasers`, `PostTeaser`).
- [Publishing](docs/publishing.md) — the game's posts on its own TikTok and YouTube through Zernio: the calendar every
  job keeps (an hour apart, 3 a trailing day on TikTok from 14:00 to 20:00 UTC, 12 on YouTube from 07:00 to 23:00), a
  post scheduled ahead held as a reservation, the upload link that keeps every video off the server, `PlanPost` (with
  `by`, a slot that waits for every platform) and `SchedulePost` under one lock and its outcomes (scheduled, busy, taken, failed); a TikTok carousel of photos in place
  of the video; the same calls on an `InternalService` for promo; no table of posts, Zernio's list is the record.
- [The posts' numbers](docs/post-stats.md) — a snapshot a day of every post, taken by the publishing module in the
  server from 08:17 UTC (`ListSnapshots`, `ListRows`), each address for the clips (`GetPostLinks`); the copy of the
  clips module's snapshots.
- [Support](docs/support.md) — suggestions, bug reports and player reports, with their photos and videos; the address
  a note is answered at, the operator's answers by email and the player's replies (a note's conversation); a letter
  the operator writes to a player first, one per flag submission, and its replies; each report told to the chat's
  judge (`PlayerReported`) and closed by its command (`DecideReport`).
- [Discord pings](docs/alerts.md) — a Discord sign-in joins the game's server, with its flag's role and its username
  as its nickname; the role follows the flag pledged, in a color of its own, and the nickname the username; the pings
  when its flag is attacked, with a link that opens the globe on the attack, and `@everyone` when a battle or a season
  starts; the operator's unlink; `LinkMembers`, which links the players in the server no sign-in linked and names
  every member; off by default.
- [The vandal squad](docs/vandalsquad.md) — what players paint on the map: each player's takes of the last hour, read
  in planet's ledger once a minute, the tiles it still holds split into drawings, each read as a swastika by rules taken
  from real ones (and the samples they are tested on), only one that shows on another flag's ground flagged, at Warn
  and in `ListGraffiti`, kept in postgres; no revert, ban or push yet.
- [Clip videos](docs/clip-videos.md) — the videos the operator posts by hand: uploaded and downloaded on the backoffice,
  the files in the bucket under `clips/videos/`, a row each in `clips.videos`.
- [The players a clip names](docs/clip-appearances.md) — a run's cast, numbers and names and never an account; who
  each post names; the opt-out, kept by clips and not by player; the public read the game tells a player from.
- [The names a post may show](docs/clip-name-verdicts.md) — a Claude Haiku judge leaves out of every cast a name
  TikTok and YouTube forbid (a terrorist group, a hate code, a mass killer); each name judged once, as it is set
  (`PlayerNamed`) or in a cast, again on a rename; the operator's verdict; no names with no key; `GetShownNames`, the
  public read of the names a player's video may show.

## Operator tools and jobs

- [Operator tools (`AdminService`)](docs/admin.md) — the console and admin listeners; `GrantCharges`,
  `ReleaseAllegiance`, `GetReplay`, `GetLedger`, `SwitchBoard`, manual bans and warnings, a player's Discord link,
  linking the Discord members.
- [Claude as a connector](docs/connector.md) — every admin method is a tool Claude calls over MCP, read from the
  admin protos (a method's comment is its description); what is hidden; auth's OAuth 2.1 server and the tokens; the
  bearer path through the gate.
- [Battle teasers on TikTok and YouTube](docs/teasers-job.md) — promo decides and posts, `cmd/teasers` only renders:
  every 6 hours, each battle starting within 36 hours posted by an hour before it through publishing's
  `InternalService`, a carousel of its cards on TikTok and a slideshow on YouTube; no battle teased twice.
- [The page's translations](../frontend/docs/i18n.md#translating) — `cmd/translations`, the bookkeeping of Claude
  Code's `translate` skill, with no network and no key: the page's lines, landmass and province names and "in" each
  flag's country that are new or changed, as batch files for Haiku subagents, and their answers checked and written
  back with a fingerprint of the English.
- [The clips job](docs/clips-job.md) — `cmd/clips`: every 2 hours, picks the best clips and posts them through the
  publishing module, over ssh; the model editor, the lab (an experiment's brief at one pick a run, by a coin, its
  post marked; `CLIPS_LAB`), the carousel (an experiment drawn as photos: the three lists, its check, its words, its
  photos as TikTok's media), `clips bench` (the war report's scripts in each tone and model, posted nowhere), the stats
  and the experiments it ran.
