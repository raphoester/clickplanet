# Twenty bounded contexts, one process

- **`internal/planet/`** — the tile game: clicks, ownership, the map, the update stream, and the charges each account holds and spends (the wallet).
- **`internal/bonuses/`** — what a caught box, a right answer or a guest's first sign-in grants. It holds no charge: it pays through planet's wallet, in planet's words, which keeps what fits; it asks no hand what it holds. It hears the boxes' catches, the quizzes' answers and player's new and forgotten players, keeps the kinds progression tells it each account unlocked (its gates draw from them), and keeps rule sets of its own that the gamemaster plays by name. It knows no level. See [The bonuses](bonuses.md).
- **`internal/boxes/`** — the boxes each caller is offered, and the claim. It pays nothing: it tells each box caught, and the bonuses module draws what it holds and pays it. It hears planet's clicks, asks planet the freeze, and keeps rule sets of its own that the gamemaster plays by name. See [The boxes](boxes-module.md).
- **`internal/quizzes/`** — the quizzes each caller is asked, the quiz bank, how hard each question is and the answer log. It pays nothing: it tells each answer, and the bonuses module pays a right one. It hears planet's clicks and player's forgotten accounts, asks planet the board and the freeze, and keeps rule sets of its own that the gamemaster plays by name. See [The quizzes](quizzes-module.md).
- **`internal/flags/`** — the catalog of flags: the ones the game ships with and the ones opened since, each closed, never deleted. It tells each flag opened, closed or renamed, and each submission approved or rejected: planet keeps from those facts which flags are open, the pages hear them on its stream, and the shop refunds a flag rejected. It hears and asks no module: the shop's payments reach it as commands in its own words. See [The flags module](flags.md).
- **`internal/shop/`** — what a player pays for: time for a regional flag, on Stripe's own page (Managed Payments). It
  asks the flags module whether time may be bought, tells it each payment's time as its commands, and hears its
  rejections to refund them; no module hears or asks the shop. Off with no Stripe key. See [The shop](shop.md).
- **`internal/chat/`** — the live chat: messages, identity, retention.
- **`internal/auth/`** — who a caller is and what it has to prove before it may click: Turnstile, accounts, the click token, which accounts may operate, and how each account came. See [Auth](auth.md) and [Arrivals](arrivals.md).
- **`internal/player/`** — who an account is to the other players: the name it chose or its guest code, its color, who is playing now, and an account asked deleted. It tells the other modules what they need of an account, so none needs to hear auth. See [Player](player.md).
- **`internal/progression/`** — how far an account went: the tiles it took, its daily streak, what it did with the bonuses, its titles and the one it wears, the season finales it fought, the countries it fought for and against, and the badges drawn beside its name, its level and the kinds of bonus each level unlocks. It hears the whole game and nothing speaks its language but the browser and the operator; it tells the bonuses module what each account unlocked, in the bonuses' words. See [Progression](progression.md).
- **`internal/scoring/`** — the points accountant: the books and rounds the gamemaster opens and closes by id, each book's player tallies, and each round's results between countries. It knows no season, day or battle. See [Scoring](scoring.md).
- **`internal/support/`** — what a player sends the operator: a feature suggestion, a bug report or a report of another player. See [Support](support.md).
- **`internal/missions/`** — each account's four or five missions of the day, how far it took them, and what each pays once done; and each flag's mission of the day, which gives the flag season points. See [Missions](missions.md).
- **`internal/antibot/`** — whether a caller plays by hand: the watchdogs, the jury and the bans. It hears what callers do from planet and asks planet to refuse the ones it bans. See [Anti-bot](antibot.md).
- **`internal/gamemaster/`** — the game's meta: which rule sets planet, the antibot, the boxes, the bonuses and the quizzes play together, and when, on a schedule of entries (a battle tonight, the Final Battle), and when each season, day and battle starts and ends, told to scoring by id. It knows the sets' names, never their values: never a point. See [The gamemaster module](gamemaster.md).
- **`internal/alerts/`** — the game's Discord server: a player's Discord link, which a Discord sign-in makes, the role of the flag it plays for, and the pings when that flag is attacked (its role) or a battle or a season starts (everybody). Off by default. See [Discord pings](alerts.md).
- **`internal/promo/`** — what the game shows of a battle before it starts: each rule set's teaser cards, kept under the set's name as planet, the antibot and the other modules keep their rules, over backgrounds the operator uploads, with words filled from the set's values at each read. It posts each battle's teaser on TikTok and YouTube through publishing; a job of its own, `cmd/teasers`, in the same package, only renders the cards. See [Promo](promo.md) and [Battle teasers on TikTok and YouTube](teasers-job.md).
- **`internal/clips/`** — the clips the game posts of itself: the videos posted by hand, for the operator
  (`clips.v1.AdminService`); a run's cast, the players each post names and the opt-out, which a player reads and sets
  on the public `ClipsService`. It asks planet who took which tile, player what each may be called and publishing
  where each post is, and hears `PlayerForgotten`. Most of the clips is a job of its own, `cmd/clips`, in the same
  package (below). See [Clip videos](clip-videos.md) and [The players a clip names](clip-appearances.md).
- **`internal/publishing/`** — the game's posts on its own TikTok and YouTube, through the one posting service,
  Zernio: the calendar every job that posts keeps (how many each platform takes in a trailing day, its hours, an hour
  between two posts), the slot of a post, and the post made at it under one lock; and a snapshot a day of every post's
  numbers. It asks no module; the clips job asks it over ssh, and promo and clips on its `InternalService`, so `clips`
  and `promo` speak its language. See [Publishing](publishing.md) and [The posts' numbers](post-stats.md).
- **`internal/vandalsquad/`** — what players paint on the map: it reads planet's ledger, asks planet what the tiles
  hold now, and keeps each drawing that reads as a swastika, for the operator. It reverts and bans nobody, and hears
  `PlayerForgotten`. See [The vandal squad](vandalsquad.md).
- **`internal/mcp/` is not a context: it is the operator's connector**, every admin method as a tool Claude calls.
  It owns no data and no proto package, and reaches the modules only as the admin page does: through the admin
  listener, behind the gate. See [Claude as a connector](connector.md).

They share the process, the transport and the flags the game ships with (`shared/cpflags`), and **nothing else**. None imports another; each owns its own domain types, its own proto package, its own storage adapter (where it has one) and its own edge.

**A module never calls another module's code or reads its data in its own stack trace.** When one needs an answer from another it asks over Connect, on a loopback listener — see [Calling another module](composite-layer.md#calling-another-module). The click token's verifying key comes from `auth` once per process, through the one verifier `cpbootstrap` hands every module (`props.Sessions`): the platform's, not a module's call. `player` asks `auth` whether an account is linked and when it was made, and answers the same of a page of accounts to whoever must not ask auth (`GetAccounts`), and the address to write to an account at (`GetEmail`). `progression` asks `player` that (`GetAccounts`), `planet` which countries an account took tiles for and from, and `scoring` a closed book's table and who fought a season's finale. `chat` asks `player` who posts (a guest or not, its code drawn the first time) and, for the operator's list, how to name a page of messages' accounts; `scoring` asks `planet` what each country holds; `support` asks `player` how to name the accounts of a report and where to answer a note (`GetEmail`); and `missions` asks `planet` the flag an account pledged, what each flag holds and how big each country is, `player` whether an account is signed in (`GetAccounts`), and `scoring` the table of a day's season, for a flag mission's points (`GetTable`); `clips` asks `planet` who took which tile in a run's window (`GetLedger`), `player` the name each of those accounts may be shown under, a name just set and the names a video may show hold (`GetPlayers`) and `publishing` where each post that names a player is (`GetPostLinks`); `bonuses` asks `planet` which kinds each hand has no room for (`GetRoom`); `boxes` asks `planet` whether the map takes no play (`GetFreeze`), and no hand's room; `quizzes` asks `planet` whether the map takes no play (`GetFreeze`) and what each flag holds, for the quiz's lean (`GetTerritories`). `vandalsquad` asks `planet` what players painted (`GetLedger`), and what those tiles hold now and what touches them (`GetTiles`). `gamemaster` asks `planet`, `antibot`, `boxes`, `bonuses` and `quizzes` which rule sets they keep, and `planet` whether a battle's set makes an arena battle (`GetRuleSet`). `promo` asks `planet` which sets it keeps and what one plays (`GetRuleSetNames`, `GetRuleSet`), `boxes`, `bonuses` and `quizzes` what their set of that name plays (`GetRuleSet`), `gamemaster` what a battle plays (`GetBattle`: its set and its points' times) and the battles to come (`GetTimeline`), `scoring` what each rank of a round of those times scores (`GetRankPoints`), and `publishing` the posts that tease a battle, a teaser's slot and its post (`ListPosts`, `PlanPost`, `SchedulePost`). `alerts` asks `planet` the flag an account pledged, `player` the name it shows for an account (`GetPlayers`), and `auth` the token a Discord sign-in left to join the game's server by (`GetDiscordGrant`) and the accounts signed in with Discord (`GetDiscordUsers`); `auth` asks no other module anything. When one needs another to change something, it sends a command: `missions` commands `planet` to grant a mission's reward (`planet.v1.GrantCharges`) and `scoring` to grant a flag mission's points (`scoring.v1.GrantPoints`), `antibot` commands it to refuse or admit a caller (`planet.v1.RefuseCaller`, `AdmitCaller`); `progression` commands `bonuses` to keep the kinds a level unlocked for an account (`bonuses.v1.KeepUnlocks`) and to pay a level reached (`bonuses.v1.PayLevelUp`); `bonuses` calls `planet` to grant what a box caught, a right answer or a sign-in earned (`planet.v1.InternalService/GrantCharges`, answered at once with what the hand kept, once per grant id), and to fill every stack a level reached earns (`FillCharges`, once per grant id); `gamemaster` calls `planet`, `antibot`, `boxes`, `bonuses` and `quizzes` to play a rule set (`PlayRuleSet` on each `InternalService`), `planet` to reset the map as a season starts (`ResetMap`) and to blank it for an arena battle and put it back (`OpenArena`, `CloseArena`), and `scoring` to open and close its books and rounds (`OpenBook`, `OpenRound`, `CloseRound`, `CloseBook`), each answered at once (see [The gamemaster's calls](composite-layer.md#the-gamemasters-calls)); `chat` asks `scoring` a round's results and a book's table by id, and commands `support` to close a report its judge decided (`support.v1.DecideReport`). When one only has to say that something happened, it publishes an event on Redis Streams and never learns who listens — see [Telling other modules what happened](composite-layer.md#telling-other-modules-what-happened). `planet` publishes `TilesTaken`, `AllegiancePledged`, `BombLanded`, `Fortified`, `ShieldPlaced`, `ShieldsStruck`, and what callers did for whoever judges it (`ClickTried`, `MapRead`, `StreamOpened`), `boxes` publishes `BoxClaimed`, through its outbox, and the box facts for whoever judges them (`BoxCaught`, `BoxLapsed`, `ForeignBoxClaimed`), `quizzes` publishes `QuizAnswered`, through its outbox, `auth` publishes `AccountDeleted`, `AccountDeletionRequested`, `AccountRestored`, `SignedIn`, `SignedOut` and `DiscordJoinGranted`, `chat` publishes `MessageSent` and `AccountMuted`, `support` publishes `PlayerReported`, `player` publishes `PlayerCreated`, `PlayerNamed`, `PlayerSignedIn`, `PlayerLeaving`, `PlayerReturned` and `PlayerForgotten`, `progression` publishes `StatsChanged`, `missions` publishes `MissionsPaid`, `MissionsDone` and `FlagMissionDone`, and `gamemaster` publishes `SeasonStarted`, `DayStarted`, `BattleStarted`, `DayEnded`, `BattleEnded` and `SeasonEnded`, and `flags` publishes `FlagOpened`, `FlagClosed` and `FlagRenamed`, the first two of which `planet` hears into its catalog, and `flags` itself all three, for its stream to the pages. `player` hears `AccountDeleted`, `AccountDeletionRequested`, `AccountRestored`, `SignedIn` and `SignedOut`, and nothing else; `progression` hears `TilesTaken`, `BombLanded`, `Fortified`, `BoxClaimed`, `QuizAnswered`, `MessageSent`, `MissionsPaid`, `SeasonEnded`, `PlayerCreated`, `PlayerLeaving`, `PlayerReturned`, `PlayerForgotten` and its own `StatsChanged`; `planet` hears `PlayerForgotten`; `bonuses` hears `BoxClaimed`, `QuizAnswered`, `PlayerCreated` and `PlayerForgotten`; `boxes` hears `ClickTried`; `quizzes` hears `ClickTried` and `PlayerForgotten`; `chat` hears `AccountMuted`, `AllegiancePledged`, `BombLanded`, `Fortified`, `FlagMissionDone`, `DayEnded`, `BattleEnded`, `SeasonEnded`, `PlayerForgotten` and `PlayerReported`; `scoring` hears `TilesTaken`, `PlayerCreated`, `PlayerLeaving`, `PlayerReturned` and `PlayerForgotten`; `missions` hears `TilesTaken`, `BoxClaimed`, `QuizAnswered`, `BombLanded`, `ShieldPlaced`, `Fortified`, `ShieldsStruck`, `DayStarted`, `PlayerSignedIn`, `PlayerForgotten` and its own `MissionsPaid`; `support` hears `PlayerForgotten`; `clips` hears `PlayerNamed` and `PlayerForgotten`; `antibot` hears planet's facts about callers and the boxes' box facts; `alerts` hears `TilesTaken`, `AllegiancePledged`, `PlayerNamed`, `PlayerLeaving`, `PlayerReturned`, `PlayerForgotten`, `DiscordJoinGranted`, `BattleStarted` and `SeasonStarted`; `vandalsquad` hears `PlayerForgotten`; and `auth` itself hears its own `AccountDeleted`, for the arrivals.

**The game modules face `player` for anything about identity, and auth is player's detail.** Player hears auth's
events and tells the others in its own words (`PlayerCreated`, `PlayerNamed`, `PlayerSignedIn`, `PlayerLeaving`,
`PlayerReturned`, `PlayerForgotten`), and answers what auth says of an account (`GetAccounts`), so no game module needs auth's language.
**Only two modules speak it**: `player`, which owns that projection, and `alerts`, whose Discord join needs the
user's OAuth token, a secret auth keeps sealed and never puts on Redis (see [Discord pings](alerts.md)). Beside them,
`auth` hears its own `AccountDeleted`, the admin gate `cmd/api` builds asks it `GetOperator`, and the connector
(`mcp`) asks it `GetTokenOperator`. Until 2026-11-04
planet, chat, scoring, support, missions, alerts and clips heard `AccountDeleted`, and missions `SignedIn` and
`GetAccount`: see [Player](player.md#the-deploy-that-moved-the-others-to-players-events).

**The modules depend on each other with no cycle.** A module depends on another when its code speaks the other's proto package: it calls it, hears its events or sends it commands. `cmd/api/modules_test.go` reads every module's imports and pins it: `TestTheModulesDependOnEachOtherWithNoCycle`, `TestNothingButTheBrowserAndTheOperatorTalksToProgression`, `TestProgressionTellsTheBonusesWhatEachAccountUnlocked` (and the bonuses module speaks no progression), `TestOnlyPlayerAlertsAndTheConnectorSpeakAuthsLanguage`, `TestProgressionKnowsPlayerAndNotAuth` (the session token's key is the platform's: `cpbootstrap` fetches it, and no module dials auth for it), `TestTheBonusesSpeakPlanetsPlayersTheBoxesAndTheQuizzesLanguageAlone` (and planet never names the bonuses module), `TestTheBoxesSpeakPlanetsLanguageAlone`, `TestTheQuizzesSpeakPlanetsAndPlayersLanguageAlone`, `TestTheModulesThatPlayOrReadTheBonusesRulesSpeakItsLanguage`, `TestTheModulesThatPlayHearOrReadTheBoxesSpeakTheirLanguage` and `TestTheModulesThatPlayHearOrReadTheQuizzesSpeakTheirLanguage`. Today, 65 pairs: `auth` and `flags` depend on none; `shop` on `flags` alone, and nothing on `shop` (`TestTheShopSpeaksTheFlagsLanguageAloneAndNoModuleSpeaksTheShops`); `planet`, `chat`, `scoring`, `player` and `alerts` on `flags` too (planet hears its facts; the others read the flags the game ships with or the shape of a code, through `cpflags`, which `cmd/api/modules_test.go` counts as the flags' language, `TestTheFlagsSpeakNoModulesLanguageAndTheModulesThatCheckAFlagSpeakTheirs`); `player` on `auth`; `planet` on `player`; `boxes` on `planet`; `quizzes` on `planet` and `player`; `bonuses` on `boxes`, `planet`, `player` and `quizzes`; `antibot` on `boxes` and `planet`; `scoring` on `planet` and `player`; `support` on `player`; `gamemaster` on `antibot`, `bonuses`, `boxes`, `planet`, `quizzes` and `scoring`; `missions` on `boxes`, `gamemaster`, `planet`, `player`, `quizzes` and `scoring`; `chat` on `gamemaster`, `missions` (whose `FlagMissionDone` it announces; missions hears nothing of chat), `planet`, `player`, `scoring` and `support` (whose reports it judges, and closes with support's own command; support hears nothing of chat); `alerts` on `auth`, `gamemaster`, `planet` and `player`; `clips` on `gamemaster`, `planet`, `player`, `publishing` and `scoring` (gamemaster, publishing and scoring for the job, whose console calls sit in the same package); `progression` on `bonuses` (whose command it sends: the gamemaster plays the bonuses' sets and progression hears the gamemaster, so the bonuses module may not hear progression), `boxes`, `chat`, `gamemaster`, `missions`, `planet`, `player`, `quizzes` and `scoring`; `promo` on `bonuses`, `boxes`, `gamemaster`, `planet`, `publishing`, `quizzes` and `scoring`, and nothing on `promo`; `publishing` on none; `vandalsquad` on `planet` and `player`, and nothing on `vandalsquad`. Player and progression were one module until 2026-11-04, and that module made three cycles: see [Why a module of its own](progression.md#why-a-module-of-its-own).

**A downstream module asks its upstream in the upstream's words.** `antibot` sends planet `RefuseCaller` and `AdmitCaller`, commands planet defines in its own proto package (`proto/planet/v1/commands.proto`) and consumes into its own table. Planet never hears the antibot's vocabulary: it tells what happened and is told whom to refuse. A command is an intent in the upstream's published language, not a fact, so it does not break the rule that facts flow from where they happen — see [Telling other modules what happened](composite-layer.md#telling-other-modules-what-happened).

**Bonus boxes and quizzes were cut out of `internal/planet/`** into `bonuses`, a module of their own downstream of
planet like `missions` (step B, 2026-11-11), so that the quizzes' difficulty and the players' ratings do not grow the
biggest module. They lived inside it because what they grant is planet's charges and what carried them was planet's
stream. The cut's rule: granting is the offers', holding and spending is planet's. Step A cut the seam inside planet
with nothing on the wire, in postgres or in the game changed; step B moved the offers out with `proto/bonuses/v1`, a
stream and rule sets of their own, and turned their ports into loopback calls to planet in planet's words
(`GrantCharges`, `GetRoom`, `GetChargesOf`, `GetFreeze`, `GetTerritories`) and a subscriber of its `ClickTried`
(`GetRoom` went on 2026-11-16, when the draw stopped reading the hand).
Planet names nothing of them. See [The offers](bonuses.md) and [Offers and the wallet](offers-and-wallet.md).

**The quizzes were cut out of `bonuses`** into a module of their own (2026-11-13), the first of three cuts that leave
`bonuses` with the reward alone, so that the reward can draw by level. Progression tells it what each level unlocked, as a command: see [Levels](levels.md#the-unlocks). The quizzes tell each
answer and the offers pay a right one; a box and a quiz no longer keep apart. See [The quizzes](quizzes-module.md).

**The boxes were cut out of `bonuses`** into a module of their own (2026-11-15), the second cut. The boxes tell each box
caught and the bonuses module draws what it holds and pays it; what is left in `bonuses` is the reward. See
[The boxes](boxes-module.md).

**`internal/antibot/` was a domain library until 2026-10-26**, built by planet's click chain and called on every click: `planet` → `antibot` was the only module-to-module import. It is a module now, wired in `cmd/api` like the others, and no module imports another.

**`internal/clips/` is mostly a job, with a binary of its own** (`cmd/clips`), and a small module of the server
beside it. The job shares no process and no config with the server, and imports only `shared`: what it asks of the
server (a replay, the race, the season, and each post, which the [publishing module](publishing.md) plans and makes)
and what it gives it (who each post names) goes over ssh to the console listener, as an operator's call. **One
package, two roots**: `clips.Config` and `Fetch`, `Choose`, `Edit`, `Schedule` are the job's, `clips.ModuleConfig` and
`NewModule` the server's, and only `NewModule` is wired into `cmd/api`. The posts' numbers are the publishing
module's since 2026-11-10: they are every post's, the teasers' too, not the clips' alone. See [The clips
job](clips-job.md).

**`internal/promo/` holds a job too** (`cmd/teasers`, 2026-11-09), the same way: `JobConfig`, `ListDue` and
`PostTeasers` are the job's, `Config` and `NewModule` the server's. **The job decides nothing**: the module says which
battles are due and posts each teaser itself; the job only hands promo's teasers to the frontend's renderer
(`npm run teaser`, in the same workflow), uploads the stills, and asks promo over ssh to post them. See [Battle teasers
on TikTok and YouTube](teasers-job.md).

## A module publishes its root package and hides the rest

Every module's interior lives behind **its own `internal/`** — `internal/planet/internal/clicks`,
`internal/chat/internal/messages`, `internal/antibot/internal/jury`. Go's own
rule does the enforcing: such a package is importable only from the tree rooted
at the parent of that `internal`, so `chat` importing `planet/internal/clicks`
**does not compile**. There is no linter to run, no allowlist to maintain and
nothing to keep in sync.

So the whole of what one module may use of another is **what sits in the other's
root package**:

| module | its public API |
|---|---|
| `planet` | `Config`, `NewModule` |
| `bonuses` | `Config`, `NewModule` |
| `boxes` | `Config`, `NewModule` |
| `quizzes` | `Config`, `NewModule` |
| `chat` | `Config`, `StorageConfig`, `NewModule` |
| `auth` | `Config`, `NewModule`, `AdminConfig`, `NewAdminGate`, `ConnectorConfig`; behind the `testing` tag, `NewModuleWithFakeProviders` and the fake's types |
| `player` | `Config`, `NewModule` |
| `progression` | `Config`, `NewModule` |
| `scoring` | `Config`, `NewModule` |
| `support` | `Config`, `NewModule` |
| `flags` | `Config`, `NewModule` |
| `shop` | `Config`, `StripeConfig`, `PlansConfig`, `PriceConfig`, `NewModule` |
| `missions` | `Config`, `NewModule` |
| `antibot` | `Config`, `NewModule` |
| `gamemaster` | `Config`, `NewModule` |
| `alerts` | `Config`, `DiscordConfig`, `NewModule` |
| `promo` | `Config`, `NewModule` (the server's); `JobConfig`, `ListDue`, `PostTeasers`, `Due`, `TeasersReport`, `TeaserPosted`, `TeaserHeld`, `TeaserSkipped` (the teasers job's) |
| `publishing` | `Config`, `ZernioConfig`, `NewModule` |
| `vandalsquad` | `Config`, `NewModule` |
| `mcp` | `Config`, `NewModule` |
| `clips` | `ModuleConfig`, `NewModule` (the server's); `Config`, `ReplayConfig`, `Fetch`, `Choose`, `Edit`, `Schedule`, `Pick`, `Platform`, `PlatformsOf`, `Report`, `PlatformOutcome` |

That holds for `cmd/api` too: the composition root lists modules and cannot
reach a domain type, a storage adapter or a controller even if it wanted to. A
module's `Config` may carry a field whose *type* is internal (`planet.Config.TilesStorage`
is `inmemory_tile_storage.Config`) — koanf fills it by reflection and a caller can
still set its fields, it just cannot name the type. That is the right amount of
access: the settings are published because they are in the file, and the code
that reads them is not.

**Adding a package inside a module is therefore free, and taking one out of
`internal/` is a deliberate act** that shows up in review as exactly one moved
directory.
