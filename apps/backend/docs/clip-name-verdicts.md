# The names a post may show (`internal/clips`, `verdicts/`)

**A clip names its top fighters, and TikTok and YouTube forbid some names.** YouTube removes what names or shows a
violent extremist, criminal or terrorist group, and follows the governments' lists to know which groups count
([its policy](https://support.google.com/youtube/answer/9229472)); the UK lists the IRA. TikTok allows no such group on
the platform in any form ([its guidelines](https://www.tiktok.com/community-guidelines/en/safety-civility)). A post
taken down is a strike, and three strikes in 90 days delete the channel. On 2026-11-09 a clip named an Irish player
called after the IRA. So **every name a cast holds is judged first, and a name the platforms forbid is left out of
the cast.** The game itself shows every name: the chat, the boards and the map are not touched.

- **The judge is in the cast, not at posting** (`casts.ShownNames`, after the opt-outs, see
  [The players a clip names](clip-appearances.md)): the end card draws the names into the video before the post is
  made, and the run's files, a public artifact, hold the cast's names. A name hidden is in no cast, so in no video, no
  caption, no file and no appearance.
- **Each name is judged once** (`screen_names_usecase`): a cast's names with no verdict, or with a verdict of another
  name (the player renamed), go to the judge, and their verdicts are kept in `name_verdicts`, one row a player. The
  names already judged ask nobody. A name the judge did not answer stays hidden and is asked again at the next cast.
- **The judge is Claude Haiku 5.5** (`claude_name_judge`), through Anthropic's Go SDK: its instructions
  (`instructions.md`, embedded) list what the platforms forbid and what stays, and say **when unsure, hide**: a name
  left out costs one mention, a strike costs the channel. Hidden: an armed, terrorist or extremist group, a hate
  ideology or its codes, an attack on a people, a real person, what looks like a private person's full name, sexual
  words, drugs and strong swearing, contact details. Kept: countries and love of them, opinions that attack no one,
  jokes, mild rude words, and the names the game makes itself. The names go as JSON, each with a number, 100 a call;
  the answer is held to a schema by structured outputs (each number, `shown`, a short `reason`), at effort `medium`,
  2 minutes and one retry. A name is data, never an instruction. **No word list**: a list kept by hand goes stale,
  and the model knows the groups and the people.
- **A judge that fails fails the cast**: an error, a refusal (`stop_reason: "refusal"`; Haiku has no fallback model)
  or an answer cut short keeps nothing, and `MakeCast` answers the error. The job then runs with no cast, so its clips
  name nobody (the backend's [The clips job](clips-job.md)). **With no `ANTHROPIC_API_KEY`, no cast names anyone**
  (`claude_name_judge.Unkeyed`): the boot logs `nameJudge: false`.
- **The operator decides over the judge** (`clips.v1.AdminService`, so Claude's connector too):
  `ListNameVerdicts(hidden_only)`, the verdicts the newest first, 1,000 at most, each with its account, its name, why
  it is hidden and who decided; `SetNameVerdict(account_id, shown, reason)`, the operator's verdict of the name the
  player was judged under (`Verdict.Overruled`), `NotFound` for a name never judged. It holds until the player
  renames: the new name is judged again.
- **A deleted account is forgotten** (`player.v1.PlayerForgotten`, group `clips-verdicts-forgotten`): its row is
  deleted.
- **What it does not undo**: a clip posted before is still on TikTok and YouTube.
- **What it costs**: one call for a run's new names, a few thousand tokens of Haiku, and none when no name is new.
- **The table**: `name_verdicts(account, name, shown, reason, decided_by, decided_at)`, `decided_by` the enum
  `verdict_by` (`judge`, `operator`), migration `20261109120000_name_verdicts`.
- **Tests**: the domain (`Screen`, `Judged`, `Overruled`), the use cases on `verdicts.MemoryStore` and
  `verdicts.ScriptedJudge` (behind the `testing` tag), the judge on a fake of Anthropic's API (the request, the
  numbers answered twice or made up, the batches, a refusal, an answer cut short), the store and the list on postgres.
