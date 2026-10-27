# Anti-bot (`internal/antibot/`)

What is left after sessions. A player who solves Turnstile in a real browser and
then runs a userscript holds a genuine session, and no address- or token-based
check can tell them from a player. The signal that survives is **behavioural**.

**It is a module of its own, and it listens to planet** (`internal/antibot/module.go`, `antibot.NewModule`, listed in
`cmd/api`). Its root publishes `Config` and `NewModule` and nothing else; the watchdogs, the jury, the bans and the
evidence are under `antibot/internal/`. Planet never imports it and never names it: it tells what callers did in its
own words, and the antibot hears it, judges, and asks planet to refuse a caller in planet's words.

```
planet ─ ClickTried, MapRead, StreamOpened, BoxCaught, BoxLapsed, ForeignBoxClaimed ─▶ antibot (listener → guard)
planet ◀────────────── RefuseCaller, AdmitCaller (planet's commands, through the antibot's outbox) ── antibot (bans)
```

- **Its own block, `antiBot`** (`antibot.Config`: `enabled`, `database`, `redis`, and the watchdogs' blocks squashed
  in), its own `antibot` schema and pool, its own Redis connection, and its own `antibot.v1.AdminService`
  (`BanPlayer`, `UnbanPlayer`, `InspectPlayer`, `GetRules`, `SaveRuleSet`; see [Operator tools](admin.md)). The block
  holds the machine only: **every bound below is a rule of the set in force**, kept in postgres and edited from the
  backoffice since 2026-10-26 ([The antibot's rules](antibot-rules.md)). A bound named here by its key (`minSuspects`,
  `bans.enforce`) is that rule.
- **Off, it is never built**: no subscriber, no route, no command. Planet then refuses nobody, since nothing tells it
  to, and `antibot.v1.AdminService` 404s.
- **`listener`** is the antibot's edge for planet's facts: one subscriber per kind of fact (`antibot-clicks`,
  `antibot-map-reads`, `antibot-streams`, `antibot-boxes-caught`, `antibot-boxes-lapsed`, `antibot-boxes-foreign`),
  each turning a fact into the guard's words (`Attempted`, `Inspect`, `Committed`, `Caught`, `Missed`, `Foreign`,
  `Fetched`, `Listened`). See [What a try tells](#what-a-try-tells).
- **`guard`** is what was the package's API before: the watchdogs, the jury and the ban book behind one facade, with
  `Hearing(feeds...)` the runner that runs the listener's subscribers and stops the watchdogs, and saves their evidence,
  only once the last fact they heard is handled.
- **`bans/outbox_planet_enforcer`** is where the antibot speaks planet's language: a running ban is a `RefuseCaller`, a
  lifted or unenforced one an `AdmitCaller`. See [A ban is told](#a-ban-is-told).
- **`observer`** holds the metric names and the `antibot ban` log line, which the click edge held before.

**The guard words its opinions itself.** The edge does two things
with a watchdog's opinion — count it if it argued for the ban, and put it in the
log line — so an `Opinion` answers `Fired()` and renders itself with `String()`,
and `Verdict`, `Evidence`, `Field` and the `clear`/`suspect`/`certain` ladder stay
inside. `Examine` follows the same rule: an `Examination` carries `Reading`s whose
level and evidence are already strings, so the edge copies them onto the wire and
never compares against the ladder. The alternative shipped briefly and is what this rule is written against:
the edge held a `formatOpinion` that compared against `antibot.Clear`, reached
through `Evidence.Rule` and `Evidence.Fields`, and decided their ordering —
sixteen lines of antibot's business in the clicks package, and four exported
names to support it. **The edge owns the message and the attribute names; how one
reading words itself is the guard's.**

The config blocks are the same bargain the rest of the backend makes — the
settings are published because they are in the file, and koanf fills them by
reflection, so a caller sets `config.Retaker.Detector.MaxSpread` without ever
naming a type.

**Detection and consequence are separate, and the consequence is the boring
half.** `antibot/internal/bans` takes a scope or an account and a clock and runs a ban. It knows
nothing about tiles, reactions or what earned it, which is why the same sentence
serves three different findings and would serve a fourth.

## A ban is told

**A banned caller's click, bomb and shield are refused, with when the ban ends.** Until 2026-10-20
they were answered `OK` and dropped, so a script author learned nothing. But most of the callers the
rules banned in October 2026 were people (`stamina` on Oct 7, `cadence` on Oct 9, `roam` on Oct 10,
`rotator` on Oct 18), and for them a click that did nothing read as a bug, with no way to ask.

**The refusal is planet's, and the antibot asks for it in planet's words.** Planet keeps its own table of refused
callers (`planet.refused_callers`, the `refusals` concept) and refuses clicks, bombs and shields from it. It never
calls out on a click, and the words ban, jury and verdict are not in it. The antibot tells it, for each caller a ban
changes, until when to refuse it.

- **Planet's commands** (`proto/planet/v1/commands.proto`): `RefuseCaller{scope | account_id, until, decided_at}` and
  `AdmitCaller{scope | account_id, decided_at}`, each on its own stream. A scope and an account are two callers, so a
  guest's ban is two commands. Planet's handlers (`refuse_caller_handler`, `admit_caller_handler`, on
  `commands:planet.v1.RefuseCaller` and `commands:planet.v1.AdmitCaller`) keep **the latest
  decision on each caller by `decided_at`** (`refusals.Refusal.Decided`): a command sent again, or an older one that
  comes after a newer one on the other stream, changes nothing. The row is changed under an advisory lock on the
  caller, as allegiances are.
- **Every change of a ban tells planet, in the ban's own transaction** (`bans.Banner.change`): the flag, the extension,
  the operator's ban, the unban and the pardon it gives. The ban row and the command commit together through the
  antibot's outbox (`antibot.outbox`, `outbox_event` `planet.v1.RefuseCaller` and `planet.v1.AdmitCaller`), and
  `cpoutbox`'s relay sends it with `streams.Commands()`: a ban kept in the antibot and never refused by planet is what the outbox is for.
  `decided_at` is taken under the ban's lock, so two changes of one key are told in the order they happened.
- **A ban not enforced refuses nothing**: with the rule `bans.enforce` off, every change tells planet to admit the
  caller.
- **Each rule set taken tells planet every running ban again** (`bans.Book.Restate`): refused until its end, or
  admitted when not enforced. The first is taken at boot, before the antibot hears anything, so the bans running at a
  deploy reach planet's table with no hand migration; a set that changes `enforce` reaches planet as soon as it is taken.
- **A ban lands a moment after the click that earned it**: the fact travels to the antibot, the command waits for the
  outbox's next tick (1s) and travels back. The few clicks in between are played. Their tries still reach the jury,
  and `reflagInterval` keeps them from extending the ban.
- **`PermissionDenied` with a `planet.v1.BanRefusal` detail**: `banned_until_unix_ms`, the end of the refusal that
  ends last of the scope's and the account's (`refusals.Book.Refusal`). The VPN block answers the same code with no
  detail, which is how the client tells the two apart; an older client shows its VPN dialog. `clicks.Refusal` is the
  domain's error, and `planetrefusal.Refusal` maps it in `click_handler`, `drop_bomb_handler` and
  `place_shield_handler`. The detail kept its name: it is the wire's.
- **No reason goes out.** The client links the terms' "Play fair" section (`/terms#play-fair`). The
  rule that fired is in the `antibot ban` log line and in `InspectPlayer`, never on the wire.
- **`refusal_click` sits outside the throttle**, as the VPN block does: a refused click spends no token, or the client
  would show a 429 instead of the refusal. `refusal_drop_bomb` and `refusal_place_shield` refuse before the bomb or the
  shield is spent, so it stays in hand. Each reads two primary keys in postgres, with 1s to do it. **A refusal that
  cannot be read refuses nothing** (`log_refusals` logs why): postgres down lets a refused caller click, rather than
  every player's clicks being refused.
- **So a ban does not feed itself**: a refused try is told (`CLICK_OUTCOME_REFUSED`), and the listener hands it to no
  watchdog. The end the player was told does not move however much it clicks; only an accomplice's click can extend it
  (see the `rotator`). The ban lapses on its own, and the evidence windows have forgotten the caller by then.
- **`refused_clicks`** counts the clicks refused, in planet. It was `banned_clicks` until 2026-10-26, and
  `shadowbanned_clicks` until 2026-10-20, and the ban metrics beside it `shadowban_flagged` (`banned_callers`) and
  `shadowban_flags` (`ban_flags`): the metrics poller's history files before those days carry the old names. The
  config block was `antiBot.shadowBan` and the package `antibot/internal/shadowban`.
- `TestARefusedCallerIsToldUntilWhenAndSpendsNoToken` pins the refusal over HTTP, and the e2e
  `TestABanReachesPlanetAndTheCallerIsRefused` the whole way round, through Redis.

## A ban is kept with its reasons

**Every ban the jury passes is a row of `antibot.convictions`**, beside the `antibot ban` log line, since journald
is no page an operator watches a battle from. It is the line's facts: when, the scope, the account and the account
whose click convicted it (`alongside`, for an accomplice), the sentence, every watchdog's reading in words, and the
click summary. The tiles the line logs are left out.

- **The jury writes it** (`jury.Convictions`, `convictions.Store`), after each flag the banner accepted and in the
  same call, so a flag inside `reflagInterval` writes nothing. A write that fails is reported like a failed flag
  (`OnStateError`), and the ban stands.
- **An operator's `BanPlayer` writes none**: it has no reading to keep, and the operator knows it.
- **Read with `Guard.Convictions(since, until)`**, oldest first, which `antibot.v1.AdminService/ListConvictions` serves (see
  [The battle reads](admin.md#the-battle-reads-listconvictions-getmetrics-getrace)). Postgres directly, no copy in memory.
- **A row is never deleted**, like the bans: it holds an address as `antibot.bans` does.
  `TestABanTheJuryPassesIsListedWithWhatItReadAndAnOperatorsIsNot` pins it.

## Ten watchdogs, one jury

A `Watchdog` measures one behaviour over one caller and returns a `Verdict`:

- **`retaker`** — takes a tile back moments after losing it, over and over.
- **`sequencer`** — walks the tile ids rather than the map: 1, 2, 3, 4, on and on.
- **`metronome`** — never varies and never stops (`cadence`), sleeps a random time between clicks (`shape`), or keeps a timer's beat through its pauses (`clock`).
- **`defender`** — nearly every take is a retake, however slowly it comes.
- **`catcher`** — catches every bonus box, at once (`catch`), or claims boxes sent to somebody else (`foreign`).
- **`cohort`** — starts, paces and stops in step with other scopes, group after group.
- **`scraper`** — reads the whole map again and again, which the web app never does.
- **`churner`** — sheds its guest account for a fresh bank: many new accounts on one scope (`churn`), or one fresh account after another across a carrier's /64s (`relay`).
- **`hopper`** — aims all over the globe: click after click lands a continent away from the last.
- **`rotator`** — plays several accounts with one hand: they share the same five minutes for hours, and almost never the same second.

**Every watchdog has two levels, and that is the design.** `Certain` is a reading
no hand produces and bans on its own. `Suspect` is a reading that would ban real
players if it were trusted alone, and counts only alongside another watchdog
measuring something else. A watchdog with one level would have to sit at the
strict end and miss every bot that jitters, or at the loose end and ban humans —
which is exactly the corner the first version of this was painted into.

The `Jury` crosses them: guilty on one `Certain`, or on `jury.minSuspects`
different watchdogs at `Suspect` inside `jury.suspicionWindow`. `rotator` never
counts.

**Crossing is worth less than it looks, and the config says so.** Weak signals
only multiply confidence when they are independent, and these only half are: a
machine sweep is sequential *and* regular *and* never rests, so two suspicions
can be one behaviour counted twice. So can the most obsessed player. It still
earns its place because `sequencer` reads ids and `metronome` reads time — two
genuinely different measurements — but `minSuspects: 2` is the loosest this
should be, and 3 in practice means only `retaker`'s certain rule ever fires.

**Two readings of zeal are one suspect, since 2026-10-17.** `stamina` (hours at
pace) and `defence` (most takes win back the flag's tiles) both measure how hard a
person plays, and the keenest players read both. Together they banned two people on
2026-10-09, and on 2026-10-12 a new player on its first day: 5h50m busy of 6h, then
81 retakes in 127 takes in a tile war, chatting all through it.

- **Each rule says it reads zeal** (`Evidence.Zeal`), and the jury counts every zeal
  reading of a caller as one suspect. The mark is saved with the opinion.
- **Zeal still counts beside any other watchdog**: `sequencer` with `stamina`, or
  `cadence` with `defence`, is still a ban. Only the two together are not.
- **The cost**: a loop that reads `stamina` and `defence` and nothing else is free,
  as the player who reads the same is.
- `TestTheNewcomerOfOctober12WhoPlaysHardAndDefendsIsNotBanned` replays the player,
  and `TestWithNoAccountTheSameHandsAreBannedAsOneCaller` is `cadence` with `defence`.

**Turns never count, since 2026-10-20.** From 10-11 to 10-19 every ban but one was
`rotator` beside `defence` or beside a `cadence` at its bound. The ones checked were
children: one on a PC, one switching between two browsers on a phone, both chatting
all through a tile war. Rotation beats the rate limit; it is not a bot signal.

- **The rotator marks its reading** (`Evidence.Turns`), and the jury counts it at no
  level. It still reads, logs, shows in `InspectPlayer` and feeds `click_rotation_share`.
- **Its ring still falls with a guilty caller**: when other watchdogs find one account
  guilty, the accounts it rotates with are banned with it (`Evidence.Accomplices`).
- `TestThePhoneOfOctober19SwitchingBrowsersInATileWarIsNotBanned` and
  `TestThePCOfOctober19AtACadenceTheMorningAfterItsTurnsIsNotBanned` replay the two.

**Crossing is also what makes it fast.** The overnight sweep from the
screenshots trips no watchdog's `Certain` for a long time — `sequencer` wants two
hundred steps, `metronome` wants half an hour — but both read `Suspect` within
two minutes, and two `Suspect`s is a ban. `TestTheOvernightSweepIsCaught` pins
that it is stopped inside 180 clicks.

**And crossing is what the counter-move costs.** Shuffle the ids and `sequencer`
goes quiet; with nothing left to corroborate it, `metronome` has to reach
`Certain` alone, which means `certainFor` — thirty minutes of free sweeping,
bought for one line of the bot's code. `TestSweepingInARandomOrderStillGetsCaught`
pins that too. The answer to that is a fourth watchdog, not a looser bound on the
third: loosening `metronome` to catch it sooner is how the obsessed player gets
banned. `hopper` is one, for shuffled ids that land all over the globe:
`TestALoopAimingAllOverTheGlobeIsCaughtBeforeTheClockIsSure`.

**Every watchdog sees every click until the caller is banned, and none while it is.** Until
2026-10-20 they saw the clicks a ban was dropping too, so a ban sustained itself: `sequencer` and
`metronome` kept re-flagging it through `reflagInterval`. A ban the player is told must not move
under it, so planet refuses those clicks and the listener hands the refused tries to no watchdog (see
[A ban is told](#a-ban-is-told)). The scenario tests (`guard`) still hand every click to the jury: what they pin is
when a ban first falls. `listener`'s own scenario feeds facts through the subscriber's handler, the way planet
tells them.

## What survives a restart

**Bans, evidence and how far each stream was heard, in postgres, in the `antibot` schema.** Bans are
`antibot.bans`, one row per scope ever banned, and `antibot.account_bans`, one row per
account, read and written by `antibot/internal/bans` with no copy in memory. What
each watchdog is tracking and the jury's record of each caller — its tally and
the last opinion of every watchdog — are `antibot.evidence`, one row per section, written by
`antibot/internal/evidence`. This exists because of 2026-09-14: production
restarted 23 times, every 5-10 minutes, during a bot attack, and with the evidence
in memory no window of 10m (`suspicionWindow`), 15m (`trackWindow`) or 30m
(`certainFor`, the catcher's `trackWindow`) ever filled. Nothing was banned.
`TestALoopRestartedEveryFewMinutesIsStillCaught` replays that, both ways.

- **The antibot owns its schema, its migrations and its pool**, as a module does:
  `antiBot.database` (a `cppg.Config`, `schema: antibot`) and `antibot/internal/migrations`. Its DI sequence connects,
  migrates, plays the rule set (which tells planet the running bans), builds the guard and loads the evidence
  (`Guard.Load`), and **an error refuses the boot**. The pool and the Redis client close once `Guard.Hearing`, the
  outbox's relay returned (`cppg.CloseAfter`, `cpredis.CloseAfter`), after the last flush of
  the evidence.
- **Bans are postgres, and nothing else.** `bans.Store` is the port: `Record`
  reads one key, `Running` reads the bans running (for `banned_callers` and the restatement at boot), and `Change` runs a function on
  one key's record under `pg_advisory_xact_lock` and writes what it answers, in one
  transaction. The ladder, the reflag interval and the unban stay in Go, on the
  `Record` and the `Banner`; the store only locks, reads and writes. So a flag, a
  ban or an unban is in the table when the call returns, two processes or a restart
  cannot disagree about one, and an operator's `UnbanPlayer` needs no restart. A row
  keeps when the key was last flagged (`last_flagged_at`), not when it may be flagged
  next: the reflag interval is added in Go, so a new `reflagInterval` applies to every
  row, and a restart is no way around it. The ban's end is `expires_at` (migration
  `20261008120000` added the first and renamed `banned_until` to the second).
  `postgres_ban_store` is the adapter and `MemoryStore` (behind the `testing` tag) the fake; `bans.StoreContractSuite` runs on both,
  and pins the lock: changes to one key never overlap.
- **No click reads them.** Planet refuses from its own table, which the commands keep (see
  [A ban is told](#a-ban-is-told)). A ban the jury could not write is reported through `Observer.OnStateError` and
  the click's fact is handled anyway: the next click of the caller is judged again, and flags it again.
- **A row is never deleted.** Scopes and accounts are two `Banner`s over two
  tables, each with its own ladder. An offence is forgotten only when an operator
  lifts its ban (`UnbanPlayer`): the row is written with one offence fewer, and
  with the ban ended when it was still running.
- **Each unban is a pardon, and every pardon is kept**: a row of `antibot.pardons` or
  `antibot.account_pardons`, with when, the ban it lifted (`flags`, `offence`,
  `banned_until`) and when it ends (`expires_at`). The `Record` carries the latest one,
  read in the same query and written in the same transaction as the ban, under the same
  lock. See [Manual bans](admin.md#manual-bans-findplayers-topplayers-banplayer-unbanplayer-revertplayer-inspectplayer).
- **Evidence is the tile map's pattern.** `evidence.Persistence` is the port,
  `postgres_evidence_store` the adapter, `MemoryPersistence` (behind the `testing`
  tag) the fake. It lives in memory and is flushed every `saveInterval` (1m), with a
  10s timeout, and once more on shutdown: it changes on every click, so a write per
  click is what it cannot afford. `guard.NewInMemory` (behind the tag) builds a
  guard over the fakes.
- **Evidence is flushed whole.** One section per watchdog and one for the jury,
  each encoded by its own package (`state.go` beside it), so a watchdog's fields
  stay unexported. A flush replaces every row in one transaction, so a section
  left out (one that fails to encode) is deleted, and
  a failed flush is simply retried whole on the next tick. A section that does
  not decode at load is reported through `OnStateError` and starts that one
  watchdog empty; the others load. That is not a failed boot: a changed section
  shape after a deploy should cost one watchdog's windows, not the start.
- **Each kind of fact is heard once, though Redis delivers it at least once.** Planet tells each kind in order, each
  at a later time than the one before (`facts.Log`), so the time names the fact. The listener keeps the latest time it
  heard of each kind (`listener.Marks`, the section `heard`) and skips a fact at or before it. The marks are saved with
  the evidence, **as of the same moment**: a flush holds the listener still while it encodes every section
  (`evidence.Holder`). So after a crash, what Redis gives back that the last flush had heard is skipped, and what it had
  not is heard again; what was heard after the last flush and acknowledged is lost with that minute of evidence, as
  before. And a clean shutdown flushes after the subscribers handled their last facts (`Guard.Hearing`).
- **A clock that steps back does not reorder the facts**: the log stamps one nanosecond after the last fact until the
  wall catches up. The box and map facts are counted at the time they are heard, as they were when the guard was
  called in the click's stack: a backlog after a restart lands in the current slice.
- **`antiBot.evidence.retention` (72h) is a ceiling on top of every window**:
  evidence older than it is dropped on load and by a sweep before each flush. The
  watchdogs' own sweeps still forget at their `trackWindow`, which is far
  shorter; retention is what bounds the rows after a long outage or a
  `trackWindow` set in days. Since each flush rewrites the rows from memory,
  nothing older than retention outlives one flush.
- An entry goes when its last event does: a metronome
  run or a jury tally still being added to is kept whole, however old its start.
- **Timestamps are wall clock, and the windows stay wall clock.** A restart of
  20 seconds costs every window 20 seconds, and a caller away for an hour is away
  whether or not the process was. Stretching every window by the outage would buy
  nothing measurable and let a long outage hide a real absence.
- **Except where a gap is the signal.** `metronome` ends a run on any gap over
  `maxGap` (3s), and no restart is that short — so a gap spanning the outage
  would read as a break every time, and the half hour of `certainFor` could never
  be reached across restarts. So the outage is taken out of that one gap
  (`detect.Outage`): it runs from the last flush (`saved_at`) to the moment
  the new guard's `Run` starts — not to the load, because the boot is not over
  then, and runners start before the server listens. If what is left is still
  under `maxGap` — the caller was clicking when the process went down and again
  as soon as it came back — the run goes on. That gap is **not a sample**: it is
  stitched from two pieces, and a stitched 0.4s inside a 950ms loop would widen
  the spread like a burst. The outage is not counted in `sustained` either. A
  crash is the same, with the outage starting at the last periodic flush (the
  latest `saved_at`).
- The jury does the same for `longestGap` and `activeFor`, which only feed the
  log line: a restart is not the caller pausing, nor time it was active.
- The jury also keeps when each watchdog last reached each level, so a reading
  standing before the restart is not reported again through `OnRise` after it.
  `cohort` saves its members and rebuilds its indexes; its cached judgement is
  not saved, so a loaded member is judged again on its next click.
- The jury keeps where each account clicked last (the section `jury-accounts`, for the retention), so
  `InspectPlayer` reads an account on the scope of its latest click. Planet's ledger told it until the antibot was a
  module.
- **What this does not change:** the jury refreshes every watchdog's opinion on
  every click before it deliberates, so a saved opinion carries its words into the
  next ban line but never decides one — the verdicts come back because each
  watchdog's own evidence does.

## What each one actually measures

**`retaker`: speed is not the signal, regularity is.** Two humans fighting over a
tile are fast too, and the player clicking back at a bot is the fastest of all.
`maxSpread` — the p90-p10 of the reactions — is the bound that does the work, and
on its own it is `Suspect`. `maxMedian` on top of it is `Certain`.

The first defaults here were wrong, and the split exists because of it. They
shipped at 250ms/120ms, sized for a bot answering off the update stream; the one
actually seen in production answers at ~1s, sailed past `maxMedian` and never
flagged — while holding a spread of 138ms across twenty reactions. Under one
boolean rule that bot was invisible. It is `Suspect` now.

**Speed on one tile is not the signal, and speed on many tiles is (`roam`).** The
band missed the Bulgaria recapture bot of 2026-09-14 from the other side: 62
retakes on 61 different tiles, median ~280ms, but a p90-p10 of ~750ms, because its
retakes queue behind the throttle. A player at war is fast only on the tile its
cursor is already on. So reactions on `minTiles` different tiles with a median at
or under `roamMedian` read `Suspect` whatever the spread. The ban line names the
stronger rule (`reflex` or `roam`) and carries `tiles` either way.
`TestTheSameSpeedOnAFewTilesIsATileWar` pins the player clicking back at a bot.

- **It only reads `Suspect`, since 2026-10-11.** Production shipped `certainTiles`
  30, and on 2026-10-10 it banned three accounts behind one address on `roam` alone:
  31 reactions on 30 tiles, median 540ms. It was two players in a duel by hand. They
  swept the same two rows of tiles for 13 seconds, each taking back what the other
  took, and the other side read `roam` too. A chase along a line puts every tile
  under the cursor, and no bound tells it from the bot. So production sets no
  `certainTiles`, and `roam` bans only beside another watchdog.
  `TestAChaseAlongALineBansNobody` replays the duel.
- **The bot is still banned, by `roam` and `defender` together**, at its fortieth
  take: `TestTheRecaptureBotOfSeptember14IsCaught` replays it at production's
  bounds. It needs those 40 takes inside 10 minutes, where 30 tiles in 15 were
  enough before.
- **A duel that goes on is banned the same way.** `roam` and `defender` both read
  retakes, so in a duel they are one behaviour counted twice. At 40 takes in 10
  minutes with six in ten of them retakes, `defender` reads too, and the two make a
  ban: the replay run on past its line is banned after about 17 seconds.
- **It counts the scope, not the payer.** 2 of the 30 tiles were another account's,
  four minutes before the duel, and two accounts were banned on their next click
  with no fast retake of their own.

**`sequencer`: the step, not the size of it.** Tile ids come from the
icosahedron's vertex order and not from a grid of latitudes, so filling in a
shape by hand does not hold a constant step from one click to the next. The rule
is the share of recent steps sitting at the modal step; the modal step's *size*
is not bounded, because a constant stride of 7 is no more human than a constant
stride of 1. A modal step of zero is excluded — that is somebody leaning on one
tile, which is the throttle's problem.

This is the only watchdog whose bounds need no measuring pass. Forty clicks at a
constant step is already past anything a hand produces; two hundred is not
arguable.

**`metronome`: the median is deliberately not bounded.** The claim is never that
the caller is fast. What is measured is `maxSpread` over an unbroken run, where a
pause longer than `maxGap` ends the run and the evidence starts again from nothing.

**The run is timed off every click tried, not every click accepted.** A loop firing
just above the refill rate has some of its tries refused, unevenly, so the gaps
between the survivors are 0.95s, 1.9s, 2.85s — a spread no clock shows. That is
exactly the bot of 2026-09-14: a try every 950ms (±60ms) for twenty minutes, half
of them 429s, read `clear`. So planet tells every try, the throttled ones included
(`try_click`, outside every gate), and the listener hands each to `Attempted`; the metronome records there
and only judges in `Watch`. `TestTheSameLoopToldOnlyByItsTakesIsMissed` (`listener`) is the same loop with the 429s
left out, and it is never caught. It also helps a player: someone spam-clicking into the
throttle is timed by their own hand, not by the refill rate.

**The run is one payer's, not one scope's, since 2026-10-11.** Keyed on the scope,
the run of a network was every hand behind it on one stream. On 2026-10-09 that
banned three accounts: people clicking by hand behind one network, in a tile war.
Two or three browsers together made 4.5 tries a second, so the gaps were short, and
a short gap has a small spread: the stream read a median of 204ms and a spread of
396ms, against `maxSpread` 400ms. Each browser alone read 420 to 640ms. `defender`
read `suspect`, as it does in a tile war, and two suspects are a ban.

- **It counts the payer**: the account the click token names, or the scope when it
  names none, as `stamina` and `hopper` do. The reading still lands on the scope's
  jury record, like every watchdog's.
- **`shape` and `clock` read the same tries, so they are the payer's too.** The merge
  worked both ways: a loop behind a busy network was hidden by it. Its neighbours'
  tries widened its spread, bent the shape of its gaps and blurred where in the
  second it landed.
- **A merged run also had no end**: one hand's pause was another hand's click, so a
  busy network could hold a run past `certainFor` and read `certain` alone.
- **A try has its payer before the throttle.** The session interceptor reads the
  token outside the whole click chain, so `try_click` tells every try with its payer, and the bot of 2026-09-14 is
  still caught.
- **With no account, the hands behind a scope are still one caller**: nothing tells
  them apart. That is the deprecated mint, and a click with no valid token while
  `auth.enforce` is off.
- **A guest that takes a new account starts a new run.** That is `churner`'s to read.
- `click_gap_skew` and `click_clock_coherence` are each payer's, not each scope's.
- **The runs are saved by payer.** A section saved before this decodes with no run
  and keeps its stamina, which was the payer's already.
- **Known risk: 400ms is close to a hand.** Simulated, a hand at even 300-900ms gaps
  with no pause over `maxGap` dips under it in 1 run of 12 inside ten minutes, and
  in 1 of 5 inside thirty. Past `certainFor` one such window reads `certain`: the
  rule asks that the run lasted, not that its spread held.
- `TestHandsBehindOneNetworkAreEachJudgedAlone`,
  `TestALoopBehindABusyNetworkIsSuspectThenCertain` and
  `TestWithNoAccountANetworkIsStillOneCaller` pin the three cases, and
  `TestTheHandsBehindOneNetworkOfOctober9AreNotBanned` replays the war.

**That run is the answer to a jittered delay.** A spread test is beatable by
construction — randomise and the band widens to look human. What is not cheap to
fake is *stopping*: a person's session has breaks in it. `activeFor` and
`longestGap` still feed no rule, because deciding on them alone would ban the
genuinely obsessed; they go in the log, beside a rule that did fire.

**Stopping was cheap to fake after all, and `shape` is the answer.** The bot of
2026-09-16 slept a random 0.6-2.1s between tries and paused up to 38s every so
often: no run lasted, its spread was ~1.2s, and every watchdog read `clear` for
hours while it rotated Free Mobile /64s. What it did not fake is the *shape* of
its gaps. A sleep drawn evenly from a range sits evenly around its median; a
hand's gaps are mostly short with a long tail. The rule is the quantile skew
`(p90 + p10 - 2·p50) / (p90 - p10)` over the last `shape.clicks` gaps.

- **Its window is not the run.** A gap over `shape.maxGap` (10s) is skipped and
  ends nothing, so a pause does not reset it; a gap over `maxGap` still ends the
  `cadence` run. A gap stitched across a restart is not a sample here either.
- **Measured before it was written.** Over two days of access log, every player
  with 500 gaps read 0.34 or more over any 500-gap window, and the bot's 12
  scopes read under 0.25 in nearly all of them. Replayed through the watchdog,
  `maxSkew` 0.25 and `certainSkew` 0.15 caught all 12 in ~12 minutes and no player.
- **It ships measuring.** `maxSkew` and `certainSkew` are pointers, unset in
  production: 0 is a skew, so leaving a bound out is the only off. The sweep
  reports every payer with a full window through `Observer.OnGapSkew`, into
  `click_gap_skew`. Set the bounds from that histogram.
- **One watchdog, one opinion.** It is a rule of `metronome` and not a watchdog of
  its own because it reads the same gaps: two timing rules in two watchdogs
  could reach `Suspect` together and ban on one behaviour. The stronger level is
  reported, `cadence` on a tie, and the evidence names the rule.
- **It is beatable too**: sleep a lopsided random time and it reads like a hand.
  It buys time, like every rule here.
- A gap that never varies (p90 = p10) has no skew and is `cadence`'s.

**A timer keeps its beat through a pause, and `clock` reads that.** The bot of
2026-09-23 ran in a hidden browser tab, which fires its timers on whole seconds.
It spent the bucket in a burst, waited five to seven minutes for the refill and
came back: no run reached `minClicks`, and a second click 50-90ms after one in
five put its spread near a second. Every watchdog read `clear` for 5h27m. What
it did not change is *where in the second* each click landed: x.13s, for hours,
through every rest. The rule puts each try on a circle `clock.period` (1s) long
and reads the length of the mean vector — 1 when every try lands at the same
point of the beat, about 1/√n for a hand.

- **It reads absolute times, not gaps**, so a pause ends nothing: the last
  `clock.certainClicks` try times are kept through breaks, and in the evidence.
  A hand's error adds up from one click to the next, so it loses the beat within
  a few clicks however regular its gaps are.
- **Measured before it was written**, over the access log of 2026-09-15 to 23:
  the bot read 0.96 over 600 tries and 0.99 over 120; no other scope passed 0.22
  over 600 or 0.50 over 120, and the heaviest players (two browsers on one /64,
  74k tries in the week) stayed under 0.10 and 0.26.
- `certainFor` (30m) is for the one human left: somebody tapping along to a
  steady beat keeps it for a song, not for half an hour.
- **It ships measuring**, like `shape`: `minCoherence` and `certainCoherence` are
  pointers, unset in production, and the sweep reports each payer with a full
  window through `Observer.OnClockCoherence`, into `click_clock_coherence`.
- It is beatable in one line — add a random delay — and then the gaps are a
  random sleep, which is `shape`'s.

**A pause of five minutes was cheap to fake too, and `stamina` counts hours.** The
bot of 2026-09-28 clicked at a guest's refill rate for eight hours a night, on one
fresh account and one fresh Free Mobile /64 per night, and stopped for five minutes
every half hour. Each pause ended the `cadence` run, it jittered, it shuffled its
tiles, and every watchdog read `clear`.

- **The rule is the busy time inside `stamina.window` (6h)**: a slice of
  `stamina.slice` (10m) is busy when the payer got `stamina.clicks` (40) past the
  throttle in it. A pause shorter than about half a slice ends nothing, and no pause
  resets anything: to stay under a bound a loop has to stop for a share of every
  window, not once.
- **It counts the payer, not the scope**: the account the click token names, or the
  scope when it names none, as the throttle keys its buckets. What it measures is
  tokens spent, and tokens are the account's; seven accounts taking turns behind one
  address are seven players here. The reading still lands on the scope's jury record,
  like every watchdog's.
- **It counts in `Watch`, not `Attempted`**: a try the throttle refused spent nothing.
- **Measured before it was set**, over three days of ledger per account: the bot's two
  accounts read 6h of 6h, the heaviest player 4h50m, and 5 of 555 accounts 3h or more.
- **It only reads `Suspect`, since 2026-10-08.** Production shipped `certainBusy`
  5h30m, and in its first week it made two bans, both on people who then asked in
  the chat why their clicks did nothing. One was a new player on a phone, on its
  first evening: six hours at its linked pace, chatting all through it, and 6h of 6h,
  as the bot read. No bound tells them apart, so production sets `minBusy` 5h30m and
  no `certainBusy`, and a run at pace for hours is banned only beside another
  watchdog. The bot of 2026-09-28 reads `clear` everywhere else, so it is free again
  until one does. `TestAnEveningAtPaceAloneBansNobody` pins it.
- **Beside `defender` it is still one suspect**: both read zeal, see
  [Ten watchdogs, one jury](#ten-watchdogs-one-jury).
- `click_busy_hours` is each payer's busy time, once a sweep, for the payers that
  clicked since the last one.
- **It is its own evidence in the metronome's section**, saved with the indexes'
  slice length; a section saved under another length drops its payers on load. A
  restart costs only the clicks it missed, since slices are wall clock.
- It is a rule of `metronome` and not a watchdog for the reason `shape` is: a loop at
  pace also reads on `cadence`, and two timing watchdogs could ban on one behaviour.

**At a faster tempo the metronome reads the game as it would be at the plain one, since 2026-10-25.** Each click
report carries the tempo in force (`detect.Click.Tempo`, from planet's `ClickTried.tempo`; 0 is plain), so a rule set that
refills 3 times as fast (see [Rules](rules.md)) does not turn its
players into suspects. Measured at production's bounds, before the change, a guest clicking each click as it came
back at 3× (gaps of 1.67s and a reaction of up to 300ms) read `Certain` on `cadence` within two hours: its jitter is
under 400ms and it makes 1,080 tries in 30 minutes.

- **`cadence` and `shape` read each gap times the tempo**, and the run's length (`sustained`, kept as the sum of its
  gaps) the same way. A hand's jitter is in proportion to its gap, so at 3× it reads three times wider and clears;
  a timer at ±20ms reads ±60ms and is still `Certain` within 30 minutes, and `Suspect` as soon as at the plain pace.
  `maxGap` 7s ends a run at a real 2.3s gap at 3×. `shape`'s skew does not move with a scale. The evidence's
  durations are at the plain tempo.
- **`stamina`'s slice is busy at `clicks` times the fastest tempo it saw** (kept with the slice): a slice at 3× is
  busy at 120 clicks, so busy still means a third of the allowance or more. A player at the full pace of the battle
  is busy, as at the plain pace.
- `clock` (unset in production) reads wall time, which a tempo does not change.
- **The rest read no rate**: `retaker` a reflex, `sequencer` the steps between tiles, `defender` a share of
  retakes, `hopper` angles, `scraper` and `churner` maps read and accounts born, `rotator` 5-minute slots. At 3× they
  reach their counts sooner, never on less. `catcher` reads the delay of each catch: boxes every 1 to 2 minutes bring
  its 5 catches in 5 to 10 minutes rather than 20 to 40, on the same bounds.
- **`cohort`'s `crowd` bound is the anti-bot's own** (`certainMembers`), never a rule set's: planet's rule sets hold
  planet's words only. A countdown makes people start on the second, a carrier's NAT can put six of them on one /24,
  and a fast tempo makes them pace alike, so a faster set may need a higher bound. `chain` and `lockstep` do not move.
- A run saved before this decodes with no length, so its `sustained` starts again at the deploy; its gaps and clicks
  are kept. `tempo_test.go` pins all of it at production's bounds.

**`defender`: what is clicked, not when.** The bots of 2026-09-14 retook from a
queue behind the throttle: tiles came back 0.4s, 1.5s, 2.5s … 40s after they were
lost, one refill at a time, so the retaker's reaction window saw almost none of
it. The rule is the share of a caller's takes, over `trackWindow`, that win a
tile back for the country that lost it within `retakeWindow`. A take for another
country, a take of what the same caller took, a refused click and a no-op are not
retakes.

**It ships measuring.** `minShare` and `certainShare` default to zero, and zero
never reads anything: the watchdog only reports each caller's share once a sweep
through `Observer.OnRetakeShare`, into the `click_retake_share` histogram (a caller
held there five minutes is five samples). That is not caution for its own sake —
two people fighting over one tile retake on every click, and
`TestTwoPlayersFightingOverOneTileReadAsRetakes` pins it. Set the shares from the
histogram, and expect the tile war to be the case that decides them.

Production reads `Suspect` since 2026-09-14 (`minShare` 0.6, `minClicks` 40 over
10m): the histogram held 26 callers under 0.1 and the recapture bot at 0.7-0.8.
`certainShare` stays unset for the tile war. A bot that only answers attacks
clicks a few times a minute, so the old 60 takes in 5m never judged it at all.

**A strike on a shield is neither a loss nor a retake** (`Click.Shielded`): the
tile did not change hands. `TestAStrikeOnAShieldIsNeverARetake` and
`TestAStrikeLosesNothing` pin both halves; a tile that falls once its shields are
gone is one loss, at the take.

**`catcher`: every box, and fast.** A box is addressed to one caller and flies
a slow orbit that is rarely in view, so a person has to zoom out to orbit height
and often drag the globe round to click it, and some boxes go by unseen. A script
reads `bonus_offered` off the stream and claims at once. Over the last
`minCatches` boxes offered (5), **all of them must be caught** — one lapse clears
the caller — and the median delay from offer to claim reads `Suspect` at or under
`maxMedian` (3s) and `Certain` at or under `certainMedian` (1.5s). Neither half is
enough alone: a player already zoomed out gets lucky once, and a keen player
catches a lot.

It is the one watchdog that does not read clicks. The registry reports each box
through `bonuses.Report` — `Caught(scope, after)` from `Claim`, timed from the
offer being sent, and `Lapsed(scope)` from the sweep — and planet tells them as facts (`BoxCaught`, `BoxLapsed`,
through `bonuses/box_facts`), which the listener hands to the guard. The watchdog keeps the outcomes
and answers from them on the caller's next click, since the jury only asks on a
click. The delay includes the round trip, which only makes a person look slower.
`bonus_catch_seconds` is the same delay as a histogram, whether the antibot is on or not.

The counter-move is cheap — wait a random few seconds, or let one box in five go
— and that is fine: a bot that does either has stopped taking every box the
moment it is offered.

**`foreign`: a box that was never yours.** From 2026-09-15 to 23 a pool of
clients painting two flags from three ISPs passed each other every box they were
offered, and each member claimed it within the same second. Only the owner can
win, so the rest were refused — 13 to 81 refused claims in their worst hour,
against at most 5 for every other scope, own boxes claimed late or twice
included. The web app only claims the box on its own screen, and a box is only
sent to its own scope.

- **The registry tells a box that was never yours from one that was.** A
  claimed or lapsed token is kept `rememberSpent` (10m) with the scope it was
  offered to. A refused claim reaches `bonuses.Report.Foreign` only when the
  token was offered to another scope or to nobody it remembers: a second tab
  clicking the same box, or a claim that lands late, is the caller's own box and
  says nothing. `bonus_claims_foreign_total` counts them.
- `foreign.minClaims` inside `foreign.window` reads `Suspect`, `certainClaims`
  reads `Certain`. **It ships measuring**: zero never reads a level.
- What a person can still produce is one per change of address between the
  offer and the claim, and one per client per restart (the offers are in memory).
- It is a rule of `catcher`, not a watchdog, because both read what a caller
  does with boxes. The stronger rule is reported, `catch` on a tie.
- The counter-move is to stop sharing tokens, and then each member's own catches
  are what `catch` reads.

**`cohort`: between scopes, not within one.** Every other watchdog judges one
scope, and a scope is only as long-lived as the caller wants it to be. On
2026-09-14 a pool painted `bg` through Firefox's built-in VPN
(`2a00:8c40:f000::/36`): pairs of /64s whose first takes were milliseconds apart,
~30 tiles a minute each for ~476s, followed at once by the next pair on new /64s.
Each identity started clean and none lived long enough to read anything — zero
`antibot ban` lines all day.

Two scopes are **in step** when both have `minClicks` for one flag
(`minFlagShare`), their first clicks are within `startWindow`, their paces are
within `rateRatio`, and — once the shorter one has been quiet for `quietAfter` —
their lengths are within `lengthRatio`. The first click is timed from `Attempted`,
since a pool starts its tries together and the throttle only blurs that.

- **`Suspect` (`lockstep`)** is `minMembers` scopes in step, from anywhere. Two
  friends joining a flag war in the same second are exactly this, which is why it
  never bans alone. It usually clears on its own: people who start together do not
  stop together, and `TestAPartnerWhoLeavesClearsTheOneWhoStays` pins it.
- **`Certain` (`chain`)** is the scope's group being the `certainCohorts`th
  separate group inside `chainWindow`, painting the same flag from the same wider
  prefix (`v4Bits`/`v6Bits`, a /24 and a /44). **A person keeps their address when
  they come back and a pool does not**, so a chain of fresh scopes is the pattern
  no crowd produces. The prefix is what separates it from a raid — waves of people
  answering one link start together too, but from all over, and
  `TestARaidFromAllOverIsNeverCertain` pins that they never reach it.
- **`Certain` (`crowd`)** is `certainMembers` scopes in step in one group from one prefix.

**How a finding reaches every member.** The jury asks per click of one scope, and
nothing here is pushed to anyone. The watchdog keeps one table of every scope,
indexed by the second of its first click and by its wider prefix, and each member
answers for itself from that table on its own next click (re-judged at most every
2s). A member that has already rotated away needs no ban; it still counts as a
link in the next group's chain. `TestTheRotatingPoolIsCaught` replays the
production pool: the third group is dropped under a minute into its eight, and
the first two are left alone because nothing about them yet is more than a suspicion.

**The flag is an input here**, unlike `topCountry` in the ban line: a pool that
paints another flag to dodge this has stopped painting the one it came for. The
counter-moves that are left cost the pool something too — stagger each start
past `startWindow`, draw its identities from unrelated ranges, or vary pace and
stay length between them. `click_cohort_scopes` is the gauge of scopes in step
right now, set once a sweep through `Observer.OnCohortScopes`: a floor that never drops to zero is a pool, whether or
not its groups have chained yet.

A rule set with a `minMembers` or `certainCohorts` of 1 is refused (`cohort.Rules.Validate`), because it would read
one scope, or one group, as a pattern. The prefix bits are the config's, and `antiBot.cohort` refuses a
prefix wider than an address at boot.

**`scraper`: what a caller reads, not what it clicks.** The web app reads the map
once per page load, in 26 `GetMap` batches, and follows `ListenForEvents` after
that; the same page load opens that stream. On 2026-09-15 and 16 a script in a
real page read one map chunk after every click, a whole map every half minute,
and painted dz and bg for twenty hours. It jittered its delay, stayed under the
throttle, picked tiles off the map and never retook one: every other watchdog
read `clear`, and the catcher's lone `suspect` banned nothing.

- **The count is maps read beyond one per stream opened**, over `trackWindow`.
  `get_map_usecase` tells each read as a share of the map (`MapRead`, two bytes per tile
  in the batch), and `listen_for_events_usecase` each stream as it opens (`StreamOpened`), even one that fails. A page
  load is one of each, so sixty players behind one carrier NAT read `clear` —
  the raw read count would not. A stream reopened after a drop reads nothing,
  and the count never goes below zero.
- **A read outside the map forfeits that credit entirely.** Tile ids start at 1
  and the web app clamps its last batch, so neither bound of a real client's read
  can land outside the map — whatever batch size it picks, which is why this needs
  no agreement with the frontend's `TILES_PER_BATCH` and cannot drift from it.
  Planet decides it (`clicks.Board.OffMap`, in `MapRead.off_map`), because it owns the map and
  its bounds; the watchdog only counts. It is reported in the ban line as `offMap`.
- **`certainOffMap` (2) such reads are `Certain` on their own**, however little was
  read, under the rule name `offMap` rather than `poll`. Forfeiting the credit was
  not enough: it only ever lowers a count, so a caller that reads the map twice and
  stops sits under `minMaps` and reads `clear`. One stray read is still forgiven —
  `TestOneReadOffTheMapIsNotEnoughOnItsOwn` — but a walk overruns the end on every
  pass, so two is already a walk. Measured over the access log of 2026-09-16: the
  7 scopes with any off-map read are exactly the 7 known bots, the lowest of them
  at 4 in its worst 15 minutes, and every other scope sits at 0.
- `minMaps` (5) reads `Suspect` and `certainMaps` (15) reads `Certain`. Measured
  over 27 hours of the access log: the bots read 23 to 31 maps in their busiest
  15 minutes, and no human scope read more than 6 — six page loads, in the web
  app's own batches. The log cannot show stream opens (Caddy writes a stream's
  line when it closes), so that 6 is before any credit.
- It is counted in 30 slices of `trackWindow`, so a caller costs the same
  however fast it reads: `GetMap` is not throttled.
- **It only observes.** Reads and the streams are refused by nothing here; a
  banned caller still loads the planet and watches it.
- **The counter-move landed on 2026-09-16, ~1.5h after this shipped**: one scope
  began opening a `ListenForEvents` in the seconds before each 26-chunk read, so
  the credit netted to nothing — 7 maps against 6 streams in its worst 15 minutes,
  against a bound of 5. It jittered its clicks past the metronome too and sat one
  suspect short of a ban while it took 4,829 tiles. What it did not change was the
  walk: every one of its reads started at tile 0 and ran six tiles past the end,
  which no page load does. Measured over the same log, all 22 human scopes read
  from 1 and stopped at the last tile, and all 6 bots did neither.
  `TestAStreamBeforeEveryReadBuysNothingOffTheLattice` pins it, and
  `TestReloadingOverAndOverIsClear` pins that a real reload still costs nothing.
- **The second counter-move landed on 2026-09-16, the same day**: a scope from the
  same carrier range walked the map four times in its first half hour, then
  **stopped reading it altogether** and clicked from the copy — 349 clicks in the
  last window against zero reads. That put it at 5.00 maps in its busiest 15
  minutes, exactly `minMaps`, so the scraper read `suspect`; every other watchdog
  read `clear` and `minSuspects: 2` was never met. It took 1,651 `ps` tiles and 640
  `dz` ones, all of them painted over other flags, and was banned by hand. Caching
  the map is what `certainOffMap` answers: the reads stop, but the walk that made
  the copy is already proof. `TestTheWalkOffTheLatticeIsCaughtBeforeItPaints` and
  `TestACallerThatStopsReadingKeepsItsVerdictForTheWindow` pin it.
- The counter-move left is to walk the map exactly as the web app does — from 1,
  in its steps, clamped — which is also to stop reading it faster than a reload.
  `click_map_reads` is each clicking caller's count once a sweep, through
  `Observer.OnMapReads`.

**`churner`: accounts, not scopes.** On 2026-09-30 two callers used a fresh guest
account every minute or two, each spending its 60-click bank in 10-30s and never
clicking again: one from a Free home /64 (265 accounts in 72h, `pl`), one from
Free Mobile, at times on one /64 and at times on a new /64 for each account, all
inside one of its /32s (`dz`, 95% of the day's clears of `fr`). Every
watchdog read `clear`: none lived long enough, and `stamina` counts per account.

- **`churn` counts guest accounts born on a scope** inside `window`: first seen
  there. A person takes their account from one /64 to the next, so a phone that
  changes /64s is one account born once. **Measured before it was set**, over the
  ledger of 2026-09-28 to 10-01: 259 of 284 /64s held one account; no /64 a person
  used started more than 3 in an hour, the bots 6 to 55. `v6` reads `suspect` at 4
  and `certain` at 6. **IPv4 has its own `v4` bounds, higher** (10 and 20, not
  measured: the game sees few IPv4 scopes), because one address is often a
  carrier's NAT. Anything that is not an IPv6 scope is held to them.
- **`relay` counts takeovers**: a fresh guest account that starts within `handoff`
  (90s) of another one stopping, in the same wider prefix (`v6Bits` 32: a mobile
  carrier hands out /64s from all over its /32) and on the same flag, when the one
  that stopped lived at most `maxLife` (5m). An account is only judged once it has
  `minClicks` (20), so each fresh identity keeps a third of its bank. It reads on
  the takeovers inside `window`: `minLinks` 3, `certainLinks` 6.
- **Replayed over the same 72h** through the watchdog: every `certain` fell on
  one of the bots' lines or relays, and the bans would have dropped 28% of the
  takes and half of the clears of `fr`. `certainLinks` 4 would have banned a 3h
  `dz` account and a Free Mobile NAT address, so 6 is two steps from that.
- **A signed-in account reads `clear`**: it costs a provider identity to replace,
  and its ban would fall on it alone. A guest's ban falls on its scope too, so the
  next fresh cookie on a churning /64 is dropped from its first click.
- **One watchdog, two rules**, for the reason `shape` is a rule of `metronome`:
  both read accounts turning over, and two watchdogs could reach `suspect` together
  on one behaviour. The stronger level is reported, `churn` on a tie. It is not a
  rule of `cohort`, which keys on scopes starting together; a relay is accounts
  following one another.
- `click_scope_accounts{family}` is each scope's count once a sweep, for the
  scopes clicked on since the last; `click_relay_links` each relay's takeovers.
  Zero bounds never read, like the defender's shares.
- The counter-moves left cost the bank: keep an account past `maxLife`, wait past
  `handoff` between accounts, or draw /64s from unrelated carriers.

**`hopper`: where a caller aims, not when.** A hand paints where it is looking.
Zoomed out, the globe shows a hemisphere, so a person does jump a continent in a
second — Canada to Australia, Africa to Brazil — but not often. A loop that picks
its tiles from anywhere on the map jumps that far on most of its steps (85% for
tiles drawn evenly over the sphere).

- **A step is two tries in a row by one payer**, `minGap` (100ms) to `maxGap` (30s)
  apart, and it **hops** when the two tiles are `minAngle` (45°) or more apart. The
  share of hops over the steps of `trackWindow`, at most `certainSteps` of them,
  reads `suspect` at `minShare` once there are `minSteps`, and `certain` at
  `certainShare` once there are `certainSteps`. Zero shares never read; the sweep
  reports each payer with `minSteps` through `Observer.OnHopShare`, into
  `click_hop_share`.
- **Measured before it was set**, over the ledger of 2026-10-06 to 09 (whole
  seconds, banned accounts included): the worst 40 steps of any account held 10%
  hops with steps up to 1s apart, 12.5% up to 10s and 17.5% up to 30s. The most
  restless player made 53 hops in three days, 0.6% of its steps. 20° is too
  tight: one player's worst 40 steps read 35% at it.
- **`maxGap` is long because the throttle is.** Production refills a click every
  2.5s to 20s, so a loop that waits for each refill never clicks twice inside 2s,
  and people given time to turn the globe still stay where they paint.
- **`minGap` is the dead network.** Taps held through a stall, or through the
  Turnstile wait, arrive milliseconds apart however far apart they were made, so a
  gap that short is no step and a flush counts only its two ends.
  `TestAFlushOfHeldTapsCountsOnlyItsEnds` pins it.
- **It reads tries, like the metronome** (`Attempted`): a try the throttle refused
  was aimed by the same hand. The position is planet's — `try_click` asks `clicks.Geography.Position` and tells it in
  `ClickTried.position` — and a try off the map has none and is no step.
- **It counts the payer, not the scope**, as `stamina` does: two players behind one
  NAT painting two continents in turn are two hands. With no account there is
  nothing to tell them apart. `TestPlayersBehindOneAddressAreJudgedApart` pins both.
  The reading lands on the scope's jury record.
- **It only reads `Suspect` in production** (`minShare` 0.4, no `certainShare`).
  No bot seen so far hopped — each painted one area — so this is for the next one,
  and alone it bans nobody (`TestHoppingAloneBansNobody`). Beside `cadence` it
  stops the shuffled loop in about two minutes instead of `certainFor`.
- **A restart stitches nothing**: the evidence keeps the steps, not the last try,
  so the first try after a boot only starts the next step.
- The counter-move is to aim near the last tile, which is to paint like a person.

**`rotator`: one hand on several accounts.** The rate limit is per account, and the
scope's bucket is ten accounts wide, so a second account on the same network doubles
the clicks. On 2026-10-09 and 10, four of the top five clickers played two or three
accounts on one scope and changed account each time a bank was empty. They are
people, not scripts, and every other watchdog read `clear`.

- **The signal is two clocks.** Time is cut in slots of `slot` (5m). Two accounts
  **share a slot** when each clicked in `minSeconds` (15) different seconds of it. A
  **shared second** is one both clicked in. One hand shares many slots and almost no
  second: in the ledger, one group of three shared 101 of 118 slots and 3 of 11,700
  seconds, and two other groups 62 and 23 slots and no second.
- **The bound is on what chance gives.** Two people with `a` and `b` active seconds
  in a slot share about `a·b/slot` of them by chance. A pair is read once that sum,
  over the slots it shares inside `window` (24h), reaches `minByChance` (90), and it
  **rotates** when its shared seconds are at most `maxShare` (5%) of it. A share of
  the active seconds would not do: two slow players share few seconds too.
- **Set from those numbers.** A household of two shared about 10% of its active
  seconds and a school of 13 to 21 players about 7%, which is what chance gives two
  players at their pace; the three groups shared 0.03%, 0 and 0. So people read near
  1 and one hand under 0.01, and the bound is 20 times under the people.
- **`minByChance` is what keeps luck out.** Two people who each click in one block
  of seconds per slot can miss each other for a while. Simulated, the worst such
  pair, judged at the first moment it can be, reads once in 400 times.
- **`minSlots` (24) shared slots read `suspect`, `certainSlots` reads `certain`.**
  Two hours of shared slots, since an hour of turns happens. Production sets no
  `certainSlots`. **The jury counts neither level**: rotation is a person who cheats
  the rate limit, and beside another watchdog it banned children (see turns never
  count, above). `TestTakingTurnsAloneBansNobody` pins it.
- **It reads tries, like the metronome** (`Attempted`), and judges in `Watch`. A try
  the throttle refused was made by the same hand, and accepted clicks alone would
  let one shared bucket space two people apart.
- **Two guests are never a pair.** The guests behind a scope spend from one bank, so
  turns buy them nothing. A pair needs one signed-in account: a signed-in account
  and its guest tab are read.
- **Same flag only.** The accounts of one hand paint one country, and an account that
  paints another to dodge this has stopped helping the first.
- **Each pair is judged alone**, so a trio inside a school is read and no student
  with it: `TestTurnsInsideASchoolAreReadAndTheSchoolIsNot`.
- **The reading names the ring.** An account's reading carries the accounts it
  rotates with (`Evidence.Accomplices`), and a guilty verdict on one falls on all of
  them: see [the parts that are easy to get wrong](#the-parts-that-are-easy-to-get-wrong).
  The evidence is the ring's numbers: `accounts`, `sharedSlots`, `sharedSeconds`,
  `byChance` and `flag`.
- `click_rotation_share` is each readable pair's shared seconds over what chance
  gives, once a sweep, for the scopes clicked on since the last one
  (`Observer.OnRotationShare`). People pile up near 1.
- **Kept per scope, per account, per slot**, one bit per second: 40 bytes a slot.
  The section is saved with the slot length, and one saved under another length is
  dropped on load.
- **What it does not see**: accounts on two networks, a phone on mobile data and a
  desktop at home. Those do not share the scope's bucket either. And two people who
  pass one device back and forth each time a bank is empty read as one hand, which
  is one more reason it only corroborates.
- The counter-move is to click with two accounts in the same seconds, which takes two
  hands or a script, and a script is what the other watchdogs read.

## The parts that are easy to get wrong

**Three things are deliberately not reactions**, and each is a way to get an
honest player flagged:

- a click onto a tile the caller's own country already holds — it is a no-op,
  `Set` publishes nothing, so it neither reacts nor becomes something to react to
- a click the handler refused (invalid country, tile out of range) — it changed
  no tile
- a click planet refused (`CLICK_OUTCOME_REFUSED`) — it never reaches the jury at all

**What a try tells.** <a id="what-a-try-tells"></a>A `ClickTried` names who (`scope`, `account_id`, `linked`,
`account_created_at` as the throttle reads it), the tile, the flag, when, and **the outcome**: taken, struck, unchanged
(a no-op), throttled, refused, another flag than the account plays for, invalid (an unknown flag, a tile off the map,
two bonuses together), or failed. With it go **the flag the tile wore before** (`owner`, `click_usecase.Out.Owner`:
what the click read right before it wrote; for a try the game answered with an error, the tile's owner now) and its
position. The listener feeds the guard as far as the click went:

| outcome | `Attempted` | `Inspect` | `Committed` |
|---|---|---|---|
| refused | | | |
| throttled, other flag | yes | | |
| invalid, failed | yes | yes | |
| taken, struck, unchanged | yes | yes | yes |

An outcome it does not know is an error, which Redis retries and then buries. `Held`, `NoOp` and `Shielded` come from
the fact, because the map no longer remembers who held the tile. `Committed` is called only for a click the game
accepted. Without that split, a griefer spams a tile with deliberately invalid clicks and the next honest player to
click it looks like it is reacting to something.

**A click a shield takes is `Shielded`, and neither a no-op nor a take.** It is the click's own outcome, struck, so
`NoOp` and `Shielded` are what the rule did. It is a click: the `retaker` times it as a reaction, and the
`defender` counts it among the clicker's clicks. It is not a take: the `retaker`
records no take on the tile, and the `defender` records no loss for `Held` and never
reads it as a retake.

Note that `sequencer`, `metronome` and `hopper` ignore all of it: a bot sweeping ids walks
over tiles it already owns and over ids the handler refuses, and both are part of
the walk.

**`try_click` is planet's, the outermost step of the click chain**, outside the refusal, the allegiance and the
throttle, so every try is told whatever stopped it. It reads nothing of the antibot and refuses nothing. Planet tells
its facts through one ordered log and one sender (`facts.Log`, `facts.Relay`, every 100ms), so each stream gets them
in the order they ended: two clicks of one caller published from two requests could reach Redis the other way round.
A crash loses the tries of the last 100ms; a Redis that does not take them loses them, counted in
`events_unpublished_total`.

**The watchdogs judge only the clicks the throttle let through** (`Inspect`): a try it refused spent nothing.

**The rule `bans.enforce` is the rollout switch,** the same shape as
`auth.enforce`: false judges, logs and counts, and tells planet to refuse nobody. The
two surfaces are built for a box with no dashboard — `click_reaction_seconds` is
a histogram whose raw bucket counts show the bot band by eye, and each flag
writes one `antibot ban` log line carrying the scope, every watchdog's verdict
and numbers (**including the ones that said `clear`** — what did not fire is half
of reading a line that did), the tiles, and the country the caller painted with
most. **The address is never a metric label** — unbounded cardinality, and
personal data in every scrape. That holds for `clicks_total` too, which
`prom_click` labels by `country_id` and `status` only. Per-address click data
is in three places instead: the ledger (every take, read with `FindPlayers` and
`TopPlayers`), `InspectPlayer` (what the antibot holds on one scope), and the
Caddy access log (every request). See [Operator tools](admin.md). `topCountry` is context for a human reading the
log and never an input to a rule on its own: the client declares it, so it is changed by
editing one string, and real players paint the same flags a bot does. `cohort` only uses the flag
to group scopes that already started together — see above.

**A flag on a running ban is rare now.** `reflagInterval` is how soon a caller
already serving a ban can be judged again, but a banned caller's clicks never reach
the jury (see [A ban is told](#a-ban-is-told)): only an accomplice's click, or a scope
banned while its account is not, flags it again. A second ban after one has lapsed
is a new offence, and climbs the ladder.

**A day with no ban still says how close it came.** A ban is the only thing
`OnFlag` reports, so on 2026-09-14 a bot attack produced zero lines and zero
metrics about the watchdogs. The jury now also reports, through
`Observer.OnRise` and `Observer.OnStanding`, into `antibot_opinions_total` and
`antibot_opinions_standing`, both `{watchdog, level}`:

- **The counter counts rises, not clicks.** A rise is a watchdog's reading of a
  caller reaching a level it has not held within `jury.suspicionWindow` — the
  same window the jury expires a reading on. A reading flapping across a bound
  every click counts once a window; one that lapses and comes back counts again.
  Per-click would say how often the watchdog was asked, and a sweep sample would
  count a standing suspicion once a minute for as long as it stands.
- **The gauge is set once a jury sweep** (`sweepInterval`, 1m): how many
  callers each watchdog reads at the level now, by the jury's own rule (latest
  reading, expired past the window). Every watchdog and level is set, zero
  included, or a gauge would hold its last non-zero value forever.
- **Levels are cumulative**, like histogram buckets: `level="suspect"` includes
  every `certain`, so a caller going straight to certain rises through both, and
  `suspect − certain` is the near misses.

The level leaves as the string `Verdict.String()` gives, not as a type — the
same rule as `Opinion`: the edge puts it on a label and never compares it.
`jury.Hooks` carries the typed verdict inside the package, and `guard.New`
words it on the way out.

**The watchdogs judge a scope; a ban falls on the scope and the account.** The
evidence stays keyed on `cpipscope.Of`, so a v6 caller cannot serve a ban on one
address and click from the next in its own /64. A click carries the account its
token names (`Click.Account`), and `bans.Book` passes a ban on:

- **the account**, when the token names one, so it is refused from any scope;
- **the scope too, when there is no account or the account is a guest's**
  (`Click.SignedIn` false: the token is not linked). A guest can
  shed its account with a new cookie, and must not shed the ban with it. A
  signed-in account's ban leaves its scope alone, so a campus is not banned for one
  player on it.

**A click, a bomb and a shield are refused when the scope or the account is refused** (planet's
`refusals.Book.Refusal`, from the commands each ban sends). `TestAGuestBannedByTheJuryCannotShedItWithAFreshCookie`
and `TestABannedGuestWithAFreshCookieIsStillDropped` pin it. The `antibot ban` line
carries `account`, and `banned_callers` counts running bans on scopes and on
accounts, so a guest banned on both counts twice.

**A ban on one account of a ring falls on all of it.** The `rotator` names the
accounts one hand plays (`Evidence.Accomplices`). When a click is guilty and a
watchdog that fired named accomplices, the jury flags each of them as it flags the
clicker (`convictsOf`): the account, and its scope too when it is a guest. Each is its
own `antibot ban` line, with `alongside` set to the account whose click was judged,
since the opinions on the line are that click's. On 2026-10-09 two accounts of a trio
were banned and the third played on; `TestTheTrioThatTakesTurnsIsBannedTogether`
bans the whole trio for a sweep on one account. A lone suspicion passes to nobody.

A scope ban only bites a bot with a stable address — against a residential proxy
pool it evaporates for exactly the reason the scope's bucket does; the account ban
is what follows a guest across addresses until it drops its cookie. `cohort` is the
watchdog that reads across scopes, beside `churner`'s relay, and it is the answer to a pool that rotates
inside one range.

`inmemory_tile_storage.Owner` exists for this: one indexed read under the existing
lock, which `try_click` asks for a try the game did not answer.
