

**Game rules** (`RulesPanel`, `rules.ts`) is `planet.v1.AdminService/GetRules` and `SaveRuleSet` (the backend's
[Rules](../../backend/docs/rules.md)): every set postgres keeps, `default` first, with its refill (`3× as fast`) and its boxes.
**Edit** opens a set in a form, its name fixed; **New set** opens an empty one. The waits
are in seconds. Save asks first (`ConfirmAction`) and is off until every value is a number and the set has a name;
the server holds the rest (a refusal quotes it, and the form stays). The game plays `default`, and follows an edit
of it within a second, for every player.
