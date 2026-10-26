# Ops

`src/ops/`: `ReconcileTitles` (`TitlesPanel`, after a deploy that adds a title or changes a rule). It asks first, and
it is harmless twice.

**Game rules** (`RulesPanel`, `rules.ts`) is `planet.v1.AdminService/GetRules` and `SaveRuleSet` (the backend's
[Rules](../../backend/docs/rules.md)): every set postgres keeps, `default` first, with its tempo (`3×`), its clicks
(`60 in hand, 0.2 a second`) and its boxes, and the set playing, marked. **Edit** opens a set, its name fixed; **Copy**
opens a copy of a set, with no name. The set playing has no Edit, as on Anti-bot rules: the rules in force change only
through the schedule, and the server refuses a save of the set that plays now (`FailedPrecondition`, "A set of this
name plays now. Save it under another name.").

**The form edits every value of a set, grouped**: Clicks (the tempo with them), Boxes, What a box is (the kinds'
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

**Rule schedule** (`SchedulePanel`, `schedule.ts`) is `rules.v1.AdminService` (the backend's
[rules module](../../backend/docs/rule-schedule.md)), with planet's and the antibot's `GetRules` beside it:

- **What plays now and since when**, from `GetSchedule.playing`, and what planet and the antibot each play. When a
  module does not play the schedule's set, the page says so in red: a module refused it, since it keeps no such set.
- **The schedule**: every entry, by start, the row playing marked, an ended one muted. A row that has not ended and names a set planet or the antibot keeps none of says which
  ("No set rush in planet or the antibot.").
- **Only an entry to come has Edit and Delete.** A running entry runs to its end; to change what plays, add an entry
  at a higher priority. There is no button that changes the rules at once.
- **Add an entry** opens a form: the set (only the ones both modules keep), its start and end (`datetime-local`, the
  operator's own time) and its priority (the higher plays). Add, Save and Delete ask first (`ConfirmAction`); the server's
  refusal is quoted (a set a module lacks, an entry that started, a start in the past), and the form stays.
