

**Game rules** (`RulesPanel`, `rules.ts`) is `planet.v1.AdminService/GetRules` and `SaveRuleSet` (the backend's
[Rules](../../backend/docs/rules.md)): every set postgres keeps, `default` first, with its tempo (`3×`), its clicks
(`60 in hand, 0.2 a second`) and its boxes. **Edit** opens a set in a form, its name fixed; **New set** opens a copy
of `default` with no name, since a set holds every value. The game plays `default`, and follows an edit of it within
a second, for every player.

**The form edits every value of a set, grouped**: Clicks (the tempo with them), Boxes, What a box is (the kinds'
weights), Charges, New accounts, Quizzes (with their on switch), and the Toll, whose steps are rows to add and remove.
`GROUPS` names each value by its path in the set's JSON (`RuleSet.toJson`), so the form needs no code per value: a
duration is typed in seconds, a count is rounded to a whole one, a toll share is typed in percent. **A grouped form,
not a JSON editor**: every value has a label and a number field, so a value cannot be left out or misspelled; a test
(`edits every value of a set`) fails when the set gains a value the form does not name.

Save asks first (`ConfirmAction`) and is off until every value is a number and the set has a name. The server holds
the rest: a refusal quotes its words (`The server refused it: …`, with the value's path), and the form stays.
