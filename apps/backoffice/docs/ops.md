# Ops

`src/ops/`: `ReconcileTitles` (`TitlesPanel`, after a deploy that adds a title or changes a rule). It asks first, and
it is harmless twice.

**Game rules** (`RulesPanel`, `rules.ts`) is `planet.v1.AdminService/GetRules` and `SaveRuleSet` (the backend's
[Rules](../../backend/docs/rules.md)): every set postgres keeps, `default` first, with its map (`Open`, or `Frozen` for
a set that takes no play, `playOf`), its tempo (`3×`), its clicks
(`60 in hand, 0.2 a second`) and its boxes, and the set playing, marked. **Edit** opens a set, its name fixed; **Copy**
opens a copy of a set, with no name. The set playing has no Edit, as on Anti-bot rules: the rules in force change only
through the schedule, and the server refuses a save of the set that plays now (`FailedPrecondition`, "A set of this
name plays now. Save it under another name.").

**The form edits every value of a set, grouped**, under one switch, "Frozen: the map takes no play" (the set's
`frozen`: while it plays, planet refuses every click, bomb, shield, refill, box and quiz; see the backend's
[A frozen map](../../backend/docs/rules.md#a-frozen-map)): Clicks (the tempo with them), Boxes, What a box is (the kinds'
weights), Charges, New accounts, Quizzes (with their on switch), and the Toll, whose steps are rows to add and remove.
`GROUPS` names each value by its path in the set's rules (`Rules.toJson`), so the form needs no code per value: a
duration is typed in seconds, a count is rounded to a whole one, a toll share is typed in percent. **A grouped form,
not a JSON editor**: every value has a label and a number field, so a value cannot be left out or misspelled; a test
(`edits every value of a set`) fails when the set gains a value the form does not name.

Save asks first (`ConfirmAction`) and is off until every value is a number and the set has a name. The server holds
the rest: a refusal quotes its words (`The server refused it: …`, with the value's path), and the form stays.

**Anti-bot rules** (`AntibotRulesPanel`, `antibotRules.ts`) is `antibot.v1.AdminService/GetRules` and `SaveRuleSet`
(the backend's [antibot rules](../../backend/docs/antibot-rules.md)): every set postgres keeps, `default` first, with
whether it enforces its bans, the watchdogs it turns off (`<watchdog>.enabled` false: the jury reads them as clear), and its
crowd (`cohort.certainMembers`), and the set playing, marked. The watchdogs are the parts of `antibot.v1.Rules` that have
an `enabled` field, read off the proto, so a new one shows with no edit here.

- **A JSON editor, not a form**: a set holds about 85 values in a dozen parts, and a field each would be a page of
  inputs. The text is the set's `antibot.v1.Rules` as protojson, every value written (`emitDefaultValues`); a bound
  left unset (`metronome.shape.maxSkew`) is left out.
- **Edit** opens a set, its name fixed; **Copy** opens a copy of a set, with no name. The set playing has no Edit: the
  rules in force change only through the schedule, and the server refuses a save of a set that plays now
  (`FailedPrecondition`, "A set of this name plays now. Save it under another name.").
- **Save** asks first (`ConfirmAction`) and is off until the set has a name and the text is an `antibot.v1.Rules`
  (`Rules.fromJsonString`: a typo in a key is refused here, with protobuf's words). The server holds the bounds; its
  refusal is quoted, every reason, and the form stays.

**Rule schedule** (`SchedulePanel`, `schedule.ts`) is `gamemaster.v1.AdminService` (the backend's
[gamemaster module](../../backend/docs/gamemaster.md)), with planet's and the antibot's `GetRules` beside it:

- **What plays now, its kind and since when**, from `GetSchedule.playing`, and what planet and the antibot each
  play. When a module does not play the schedule's set, the page says so in red: a module refused it, since it
  keeps no such set.
- **The schedule**: every entry, by start, with its kind, a battle's times (`3×`) and its description (cut at 60
  characters, whole on hover), the row playing marked, an ended one muted. A row that has not ended and names a set a module keeps none of says
  which ("No set rush in planet or the antibot.").
- **Only an entry to come has Edit and Delete.** A running entry runs to its end; to change what plays, add an entry
  at a higher priority. There is no button that changes the rules at once.
- **Add an entry** opens a form: the set (only the ones both modules keep), its kind, a battle's times, its description, its start and
  end (`datetime-local`, the operator's own time) and its priority (the higher plays). Add, Save and Delete ask first
  (`ConfirmAction`); the server's refusal is quoted (a set a module lacks, an entry that started, a start in the past),
  and the form stays.
- **The description is the entry's words for the players**, plain text the game client shows behind the battle chip
  ("Triple points. A bonus box every minute."): the client cannot know a set's values, so the operator writes them.
  At most 300 characters (`MAX_DESCRIPTION`, the server's bound, which the field and `complete` hold too); it may be
  empty.
- **The kind is what the entry is to the players** (`gamemaster.v1.EntryKind`): `battle`, which the game client gives a
  war-like look, or `plain`, a change of the rules with no look. A new entry starts as a battle. The picker lists every
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
`DeleteSeason`. The operator plans a season by its break and its length; the server works out its window.

- **Every season, by number**, with its break, start, length, end and Final Battle ("2 h, rush × 3 from …", or none),
  and whether it is done, running or to come; the running one marked, a done one muted.
- **The form edits the season's Final Battle** (`FinaleDraft`): "Ends with a final battle", then its length in days and
  hours, its rule set, its times (1 to 10) and its words for the players. Add and Save are off until the set is named
  and the times are whole. **Every Save sends the finale the form holds**, so an edit of the length keeps it; unticked,
  the season has none. Once it started, the length and the finale are locked: the server refuses them
  (`FailedPrecondition`).
- **The form takes the break and the length in days and hours** and shows, as they are typed, the window they give
  (`windowOf`: after the season before's end, or season 0's own start). A new season has no break and 28 days.
- **A season that has not ended has Edit**; a running one keeps its break (the fields are locked). A duration left as
  it was is sent as kept, to the nanosecond. **Only the last season, before it starts, has Delete.**
- Add, Save and Delete ask first (`ConfirmAction`); the server's refusal is quoted, and the form stays.
