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
- **A name is judged as it is set** (`player.v1.PlayerNamed`, group `clips-verdicts-named`, `player_named_subscriber`,
  `judge_name_usecase`): a name drawn at a first sign-in (an adjective, an animal and two digits, which the instructions
  keep), one the player chose, and an operator's rename. The event holds no name: the use case asks player the name the
  account holds now (`GetPlayers`, through `rpc_player_names`; a guest or an account asked deleted has none) and hands
  it to `screen_names_usecase`, so a name judged already asks nobody, and two events close together judge the latest
  once. A player who turned the clips off is judged too: the verdict is of the name. **A judge that fails fails the
  event**, which is tried again in place (`redis.maxDeliveries`, about four minutes) and then buried; the name stays
  hidden and is asked again at its next cast or its next rename. The subscriber gives each event a minute, a call of the
  judge and its retry. **With no key the module does not hear the event**: no judge would ever answer, and every event
  would be buried. The names set before 2026-11-11 are judged only in a cast or at their next rename: there is no
  backfill.
- **The judge is Claude Haiku 5.5** (`claude_name_judge`), through Anthropic's Go SDK: its instructions
  (`instructions.md`, embedded) list what the platforms forbid and what stays, and say **when unsure, hide**: a name
  left out costs one mention, a strike costs the channel. Hidden: an armed, terrorist or extremist group, a hate
  ideology or its codes, an attack on a people, a terrorist, a mass killer or the leader of a genocide, sexual words,
  drugs and strong swearing, contact details. Kept: countries and love of them, opinions that attack no one, any
  other person (a hero, a politician, a star, a first and a family name), jokes, mild rude words, and the names the
  game makes itself. **Only what the platforms forbid is hidden**: until 2026-11-10 the instructions also hid every
  real person and every name that read as a private person's full name, which the platforms do not forbid, and the
  judge hid "RAYNA KNYAGINYA", a Bulgarian national heroine. The names go as JSON, each with a number, 100 a call;
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
- **`clips.v1.ClipsService/GetShownNames(account_ids)` is the public read of the names a player's own video may show**
  (the frontend's video names its maker and its allies, and the welcome line a newcomer's recruiter). It answers each
  account whose verdict is shown, by the judge or the operator, **of the name it holds now**, and that did not turn the
  clips off, with that name, once each, in the order asked. Every other account is left out: a guest, one unknown,
  deleted or asked deleted, one never judged, one judged hidden, one judged under a name it left, one kept out. **It
  asks the judge nothing**: a read never writes, and a name not judged yet is hidden until it is. `shown_names_query`
  asks player the names held now (the same `rpc_player_names`), then one statement joins them on `name_verdicts` and
  `opt_outs`; `TestTheReadShowsTheNamesACastShowsWhenTheJudgeAnswersNothing` pins it against `casts.ShownNames`, the
  same rule on the write side. **It needs no token**: the session interceptor does not list it, and names are public.
  At most 100 ids a call (`get_shown_names_handler.MaxAsked`), more is `InvalidArgument` (`ErrTooMany`), as is an id
  that is not an account. `NO_SIDE_EFFECTS`, and `no-store`: a rename, an opt-out or the operator's verdict holds at
  the next read. No rate limit, as player's `GetNameplates`: a page is one call to player and one statement. Caddy
  routes it with the rest of `clips.v1.ClipsService`.
- **A deleted account is forgotten** (`player.v1.PlayerForgotten`, group `clips-verdicts-forgotten`): its row is
  deleted.
- **What it does not undo**: a clip posted before is still on TikTok and YouTube.
- **What it costs**: one call for a run's new names, a few thousand tokens of Haiku, and none when no name is new; one
  call of one name for each name set.
- **The table**: `name_verdicts(account, name, shown, reason, decided_by, decided_at)`, `decided_by` the enum
  `verdict_by` (`judge`, `operator`), migration `20261109120000_name_verdicts`.
- **Tests**: the domain (`Screen`, `Judged`, `Overruled`), the use cases on `verdicts.MemoryStore` and
  `verdicts.ScriptedJudge` (behind the `testing` tag), the judge on a fake of Anthropic's API (the request, the
  numbers answered twice or made up, the batches, a refusal, an answer cut short), the store and the list on postgres,
  the subscriber of `PlayerNamed` on a fake use case, `GetShownNames`' handler on a fake query and its query on
  postgres, and the module booted, read with no token.
