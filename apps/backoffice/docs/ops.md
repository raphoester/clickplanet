# Ops

`src/ops/`: `ReconcileTitles` (`TitlesPanel`, after a deploy that adds a title or changes a rule). It asks first, and
it is harmless twice.

**Anti-bot rules** (`AntibotRulesPanel`, `antibotRules.ts`) is `antibot.v1.AdminService/GetRules` and `SaveRuleSet`
(the backend's [antibot rules](../../backend/docs/antibot-rules.md)): every set postgres keeps, `default` first, with
whether it enforces its bans, the watchdogs it turns off (`<watchdog>.enabled` false: the jury reads them as clear), and its
crowd (`cohort.certainMembers`), and the set playing, marked. The watchdogs are the parts of `antibot.v1.Rules` that have
an `enabled` field, read off the proto, so a new one shows with no edit here.

- **A JSON editor, not a form**: a set holds about 85 values in a dozen parts, and a field each would be a page of
  inputs. The text is the set's `antibot.v1.Rules` as protojson, every value written (`emitDefaultValues`); a bound
  left unset (`metronome.shape.maxSkew`) is left out.
- **Edit** opens a set, its name fixed; **New set** opens a copy of `default`, with no name.
- **Save** asks first (`ConfirmAction`) and is off until the set has a name and the text is an `antibot.v1.Rules`
  (`Rules.fromJsonString`: a typo in a key is refused here, with protobuf's words). The server holds the bounds; its
  refusal is quoted, every reason, and the form stays.
