# Ops

**Timeline** (`TimelinePanel`, `timeline.ts`) is what the gamemaster does next, read from `GetSchedule`, `GetSeasons`
(with each season's days) and planet's `GetRules`, and worked out on the page:

- **Four lanes of blocks** over the range picked (1 day, 3 days, 1 week, 4 weeks, from a little before now), with a
  red line at now: Seasons (and the breaks between them), Rules in force, Battles (the entries and the seasons'
  Final Battles) and Days. Hovering a block gives its words and times.
- **Rules in force follows the gamemaster's rule**: a season's finale above all, else the running entry of the highest
  priority (at one priority, the one made last), else `default`, which is left blank. A set planet keeps as frozen is
  drawn as frozen.
- **The list below is every time something happens in the range**, the next one marked, each with how long until
  it, and what happens then in the gamemaster's order: a day closes, a battle ends, a season ends, a season starts,
  the map is reset (every season but the first), a battle starts, a rule set plays.
- **It reads, and changes nothing.** Reload reads again; the times move on by themselves every 30 s.

`src/ops/`: `ReconcileTitles` (`TitlesPanel`, after a deploy that adds a title or changes a rule). It asks first, and
it is harmless twice.

**Both rule pages edit a set the same way.** The list of sets marks the one playing ("Playing now") and shows each set
as it plays: its own values, and `default`'s for the rest (`playedOf`, `overlay.ts`, as the backend plays them).
**Edit** opens a set, its name fixed, the one playing too; **Copy** opens a copy of a set, with no name. A copy of
`default` holds nothing of its own, so it plays as `default` until a value is typed. A save of the set playing takes
over at once, and the save's question says so ("It plays now: it takes over at once."), for an Edit and for a Copy
given its name. The form takes the page's place, with "← All sets" back.

- **What the form changes shows as it is typed** (`changesOf`, `components/Changes`): a value changed is outlined and
  says what it was ("2 min · was 4 min"), and the bar at the bottom of the window counts the changes. A duration says
  what it reads as beside its seconds ("240" → "4 min", `secondsWord`); "3" and "3.0" are one value.
- **`default` holds every value; another set holds only what it changes** (the backend's
  [Rules](../../backend/docs/rules.md#a-rule-set)). In a set other than `default`, an empty field plays default's value,
  which the field shows ("default: 60", and as its placeholder), and "Use default" empties a field the set holds; a
  switch is a choice of "Default (yes)", "Yes" and "No"; the toll is default's until "A toll of its own" is ticked,
  which starts from default's steps. A save sends only what the set holds, zeros included. In `default`, a switch is a
  box and every value it holds stays needed.
- **Save** sits in that bar, always in reach. It asks first (`ConfirmAction`) with **the list of changes**, each
  value's place, what it held and what it will hold. It is off until the set has a name and every value is a number,
  and, for an Edit, until something changed; a Copy may save as it came, under its new name. The server holds the
  rest: a refusal quotes its words (`The server refused it: …`, with the value's path), and the form stays.

**Game rules** (`RulesPanel`, `rules.ts`) is `planet.v1.AdminService/GetRules` and `SaveRuleSet` (the backend's
[Rules](../../backend/docs/rules.md)): every set postgres keeps, `default` first, with its map (`Open`, or `Frozen` for
a set that takes no play, `playOf`), its tempo (`3×`), its clicks (`60 in hand, 0.2 a second`) and its boxes.

**The form edits every value of a set, grouped**, under two switches, "Frozen: the map takes no play" (the set's
`frozen`: while it plays, planet refuses every click, bomb, shield, refill, box and quiz; see the backend's
[A frozen map](../../backend/docs/rules.md#a-frozen-map)), and "Flags locked: no player changes country" (the set's
`allegianceLocked`, for a battle's set: see the backend's [Allegiance](../../backend/docs/allegiance.md)): Clicks (the
tempo with them), Boxes, What a box is (the kinds'
weights), Charges, New accounts (with "Refill at first sign-in", the set's `bonuses.signInRefill`: see the backend's
[A guest's first sign-in](../../backend/docs/sign-in-refill.md)), Quizzes (with their on switch), and the Toll, whose steps are rows to add and remove.
`GROUPS` names each value by its path in the set's rules (`Rules.toJson`), so the form needs no code per value: a
duration is typed in seconds, a count is rounded to a whole one, a toll share is typed in percent. Every value has a
label and a number field, so a value cannot be left out or misspelled; a test (`edits every value of a set`) fails when
the set gains a value the form does not name.

**Anti-bot rules** (`AntibotRulesPanel`, `antibotRules.ts`) is `antibot.v1.AdminService/GetRules` and `SaveRuleSet`
(the backend's [antibot rules](../../backend/docs/antibot-rules.md)): every set postgres keeps, `default` first, with
whether it enforces its bans, the watchdogs it turns off (`<watchdog>.enabled` false: the jury reads them as clear), and its
crowd (`cohort.certainMembers`). The watchdogs are the parts of `antibot.v1.Rules` that have an `enabled` field, read
off the proto, so a new one shows with no edit here.

- **A form read off the proto** (`PARTS`, `LEAVES`): a set holds 94 values in a dozen parts, so each part (Bans,
  Jury, each watchdog) is folded under one line that says whether it is on and how many of its values the form
  changed. Inside, its switch first, then a field per value, labelled from its name ("minReactions" → "Min
  reactions", the whole place as the field's name: "Metronome · Min reactions"), and its inner parts (Shape, Clock…)
  framed. A value the proto gains shows with no edit here; a kind of value the form has no field for (an enum, a map,
  a string) fails loud.
- **A value as it is typed**: a bool is a box in `default` and a choice with "Default" elsewhere, a count is rounded
  to a whole one, a duration is in seconds, and the ban ladder (`bans.banDurations`) is steps to add and remove. A value
  left empty is left unset: another set judges with default's, which the field shows. In `default`, a value it held
  stays needed, and one it left unset (`metronome.shape.maxSkew`, which then never reads) may stay so; the form says
  which value is missing or no number (`flawOf`). A part's line shows its switch as the set plays it, "(default)" when
  it is default's.

**Rule schedule** (`SchedulePanel`, `schedule.ts`) is `gamemaster.v1.AdminService` (the backend's
[gamemaster module](../../backend/docs/gamemaster.md)), with planet's and the antibot's `GetRules` beside it:

- **What plays now, its kind and since when**, from `GetSchedule.playing`, and what planet and the antibot each
  play. When a module does not play the schedule's set, the page says so in red: a module refused it, since it
  keeps no such set.
- **The schedule**: every entry, by start, with its kind, a battle's times (`3×`) and its description (cut at 60
  characters, whole on hover), the row playing marked, an ended one muted. A row that has not ended and names a set a module keeps none of says
  which ("No set rush in planet or the antibot.").
- **Only an entry to come has Edit and Delete.** A running entry runs to its end; to change what plays, add an entry
  at a higher priority. There is no button that changes which set plays at once; the set playing is edited on its rules page.
- **Add an entry** opens a form: the set (only the ones both modules keep), its kind, a battle's times, its description, its start and
  end (`datetime-local`, in the operator's own time zone, named above the fields) and its priority (the higher plays).
  For a battle, the form says when days end ("Days end at 23:00.", from `GetSeasons.daily_cutoff`) and warns in red of
  one that neither starts nor ends then (`offCutoffOf`); Add stays on. Add, Save and Delete ask first
  (`ConfirmAction`); the server's refusal is quoted, its times in the operator's own (a set a module lacks, an entry
  that started, a start in the past), and the form stays.
- **The description is the entry's words for the players**, plain text the game client shows behind the battle chip
  ("Triple points. A bonus box every minute."): the client cannot know a set's values, so the operator writes them.
  At most 300 characters (`MAX_DESCRIPTION`, the server's bound, which the field and `complete` hold too); it may be
  empty.
- **The kind is what the entry is to the players** (`gamemaster.v1.EntryKind`): `battle`, which the game client gives a
  war-like look, `halloween`, which it dresses for Halloween, or `plain`, a change of the rules with no look. A new entry
  starts as a battle. **To dress the game for Halloween**, add an entry of kind halloween that plays the set that would
  play anyway (`default`), at a priority under any battle: it scores nothing, and the Final Battle plays over it. The picker lists every
  kind the proto names but unspecified (`kinds`, `wordOf`), so a kind added to the contract shows with no edit here.
- **A season's Final Battle is in the list too** (`linesOf`, `FinaleLine`), from `GetSchedule.finales`: its set, kind
  battle, its times and words, "above all" for its priority, and "Season N's final battle" where an entry has its
  buttons. It comes from its season and is changed on the Seasons page only. While it plays, "Playing" names it
  (`Playing.finale_season`) and its row is marked.
- **A battle's times** (`Entry.times`) are what its points are worth: scoring scores the battle's window as a round of
  its own, its points times this. A whole number from 1 to 10 (`MAX_TIMES`, the server's bound); a new battle starts at
  3. A plain entry has no round: the field is hidden and the page sends 0 (`timesOf`). Scoring keeps no rule set:
  only planet and the antibot play the schedule's set.

**Seasons** (`SeasonsPanel`, `seasons.ts`) is `gamemaster.v1.AdminService/GetSeasons`, `AddSeason`, `EditSeason` and
`DeleteSeason`, with planet's and the antibot's `GetRules` beside it. **The operator picks times, never durations**
(the owner, 2026-11-01: "I don't want to have to do mental calculations"): the page turns them into the break and the
length the server keeps.

- **Every season, by number**, with its break, start, length, end and Final Battle ("2 h, rush × 3 from …", or none),
  and whether it is done, running or to come; the running one marked, a done one muted.
- **The form takes the season's start and end** (`datetime-local`, in the operator's own time zone, named above the
  fields), and says what they give as they are typed: "27 d 6 h long, starting 16 h after season 0 ends"
  (`spansOf`). The break is the start less the end of the season before (season 0's own start less its break); the
  length is the end less the start. **A time left as it was sends its duration as kept, to the nanosecond**; a start
  moved alone keeps the end where it was. A new season starts as the last one ends, and lasts 28 days.
- **"Days end at 23:00."**: the daily cutoff in the operator's time on the season's end day, from
  `GetSeasons.daily_cutoff` (`cutoff.ts`). A server that sends none shows no line and no cutoff warning.
- **The form edits the season's Final Battle** (`FinaleDraft`): "Ends with a final battle", its length in hours, its
  rule set (a list of the sets both modules keep, `playable`, as the schedule's), its times (1 to 10) and its words for
  the players, then **its times as they are typed** ("Final battle from … to …", `finaleWindowOf`). Add and Save are
  off until both times are filled, the set is picked and the times are whole. **Every Save sends the finale the form
  holds**, so an edit of the end keeps it; unticked, the season has none.
- **What the server will refuse shows in red as it is typed** (`flawsOf`): a start before the season before ends, a
  start or a moved end before now, an end before the start, a Final Battle longer than the season, and **a Final
  Battle that neither starts nor ends at the cutoff** ("The final battle runs 15:00 to 19:00: it must start or end at
  23:00, when a day ends."). Save stays on: the server holds the rules.
- **A season that has not ended has Edit**; a running one keeps its start (the field is locked). Once its Final
  Battle started, the end and the finale are locked: the server refuses them (`FailedPrecondition`). **Only the last
  season, before it starts, has Delete.**
- Add, Save and Delete ask first (`ConfirmAction`), with the times in the operator's time ("Run season 1 from … to …
  (27 d 6 h long, starting 16 h after season 0 ends), ending with a final battle of 4 h, urf × 3 from …?"); the
  server's refusal is quoted, its times in the operator's own, and the form stays.
