# The offers (`internal/bonuses/`)

**The bonuses module offers the boxes and the quizzes, and grants what a caught box or a right answer is worth.** It
holds no charge: the charges a player holds and spends are planet's wallet ([Charges](charges.md)). The rule of the
cut: **granting is the offers', holding and spending is planet's.** The module left planet on 2026-11-11 (step B of
[Offers and the wallet](offers-and-wallet.md)), so that the quizzes' difficulty and the players' ratings (step C) do not
grow planet, the biggest module.

It is downstream of planet like [missions](missions.md): it hears planet's clicks, asks planet's wallet, and planet
knows nothing of it. **It depends on planet and player alone**: `cmd/api/modules_test.go` pins it
(`TestTheBonusesSpeakPlanetsAndPlayersLanguageAlone`, which also pins that planet names no `bonuses`).

## What it holds

| package | what |
|---|---|
| `internal/bonuses` | the registry (`bonus.go`, `quiz.go`, `welcome.go`, `entrant.go`), the ports it asks planet through (`wallet.go`), the box facts (`box_facts`), its rules (`rules.go`, `message.go`) and the use cases `claim_bonus`, `open_quiz`, `answer_quiz`, `grant_sign_in_refill` and `listen_for_events`, moved from `planet/internal/bonuses` as they were |
| `internal/quizzes` | the bank and its draw, moved from `planet/internal/quizzes`; it embeds `generated/quiz/bank.json` ([The quiz bank](../../../docs/quiz-bank.md)) |
| `internal/rules` | its rule sets: snapshots, sets, parts, the set in force, the choice, their postgres stores and the play and save use cases, the same machinery as planet's and the antibot's |
| `internal/facts` | its own facts log and relay, for the box facts |
| `internal/planetwords` | planet's words for a kind of charge and a hand (`planet.v1.BonusKind`, `ChargesHeld`) |
| `internal/subscribers` | the clicks heard from planet, the new players heard from player |
| `internal/bonusesv1controller` | the handlers, the session interceptor and reader, the freeze interceptor |

Its postgres schema is `bonuses` (`bonuses.rule_sets`, `bonuses.rule_set_choice`); nothing of a box or a quiz is written
there, as nothing was in planet's. Its config is the `bonuses:` block: `database`, `redis`, and `offers.forgetAfter`
(5 min, a caller with no stream open is forgotten) and `offers.sweepInterval` (1 s), which were planet's top-level
`bonus:` block.

## The wire (`proto/bonuses/v1/`)

- **`BonusService`**, public, routed by Caddy (`/bonuses.v1.BonusService/*`):
  - `ClaimBonus`, `OpenQuiz`, `AnswerQuiz`: the requests and answers planet's were, in planet's words for what they
    grant (`planet.v1.BonusKind`, `planet.v1.ChargesHeld`). `NotFound` for a box or a quiz not offered to the caller,
    gone or taken; `FailedPrecondition` with `bonuses.v1.FrozenRefusal` while frozen (below). The click token is checked
    as planet checked it (`cpconnect.Attested`, counted in `bonus_session_checks`).
  - `ListenForEvents`: a stream of its own, `BonusEvent` — `heartbeat`, `bonus_offered` and `quiz_offered` (to the caller
    alone), `bonus_taken` (to every stream, with `quiz_subject_country_id` for a quiz won) and `rules_changed` (the offers
    took another set). A heartbeat at once and every `httpServer.streamHeartbeat`, a clean end on the drain
    ([The live streams](live-streams.md)). Its session reader refuses nothing: a signed-in account's stream attends as
    that account, any other as its address, which every guest behind it shares (`bonuses.EntrantOf`). **An offer goes
    only to a caller with this stream open** (`caller.watching()`).
  - `GetGrantRules`: what the page shows of the offers' rules, `sign_in_refill` today. A GET no cache keeps;
    `rules_changed` sends the page to read it again.
- **Events** (`events.proto`): `BonusClaimed`, `QuizAnswered`, and the box facts `BoxCaught`, `BoxLapsed`,
  `ForeignBoxClaimed`, with planet's fields and numbers, so a payload planet told reads as the new type.
- **`InternalService`**: `PlayRuleSet` (the gamemaster's; the latest `decided_at` wins; `FailedPrecondition` for a set
  the offers keep none of or cannot play), `GetRuleSetNames` and `GetRuleSet` (default's values with the set's own over
  them; `NotFound`), for the gamemaster and promo.
- **`AdminService`**: `GetRules` and `SaveRuleSet`, which the backoffice's Box & quiz rules page calls and Claude calls as
  `bonuses_get_rules` and `bonuses_save_rule_set` ([Claude as a connector](connector.md)).

## What it asks of planet, in planet's words

| port | planet's answer | when |
|---|---|---|
| activity (`Registry.Clicked`) | `planet.v1.ClickTried`, group `bonuses-clicks` | every click planet tried |
| `Room.Full` | `InternalService/GetRoom` | once per sweep for each caller due, outside the registry's lock |
| `Wallet.Pay` | `InternalService/GrantCharges` | each box caught, right answer, sign-in refill |
| `Wallet.Hand` | `InternalService/GetChargesOf` | each wrong answer |
| `quizzes.Shares` | `InternalService/GetTerritories` | every 10 s, by a runner |
| the map's freeze | `InternalService/GetFreeze` | each claim, opening and answer |

- **Activity.** `click_tried_subscriber` keeps the outcomes `TAKEN`, `STRUCK` and `UNCHANGED`, exactly what planet's
  `bonus_click` heard inside the click chain before the move (see [What `Clicked` hears](offers-and-wallet.md#what-clicked-hears)).
  A click carries its caller (scope, account, linked), `tried_at`, `dry` (the bank left empty, told by planet's
  `try_click` since 2026-11-11) and the account's age, read off its UUIDv7 id (`cpsession.AccountID.CreatedAt`).
  - **The delay.** Planet's `facts.Relay` sends what its log took every 100 ms in one pipelined `XADD`, and the group's
    blocking read returns as soon as an entry lands: a click is heard about 0.1 s after it was tried, a busy 100 ms in
    one batch, in order. The registry times a click by `tried_at`, so `activeWithin` (2 min) counts from the try as
    before; a click heard after `activeWithin` (a backlog read at a boot) is no sign of play and is dropped. A
    newcomer's early quiz is due when its dry click is heard and offered at the next sweep: at most about 0.1 s later
    than when the offers sat in planet.
  - A `ClickTried` Redis refused (counted in `events_unpublished_total`) is a click the offers never hear, as the
    antibot never does: at worst a newcomer's dry click, whose early quiz waits for its next dry click.
- **Room.** Planet answers, for each account asked, the kinds its hand has no room for: a full stack, a charge held
  where one is the most, and **a bomb while planet's `charges.bomb.off` is on**. So `bomb.off` is planet's rule alone:
  the offers never offer a bomb then, since no hand has room for one. An account with room for everything is left out of
  the answer. An error loses the slot, as before ([Room](offers-and-wallet.md#room)).
- **Pay.** Planet grants once per grant id and answers what the stack kept and the hand after. A box's or an answer's id
  is its token's (`OfferGrantOf`) and planet keeps it 10 minutes in memory (`lasting` unset), since the offer itself
  lives only in memory and both modules share one process: a retry never outlives a restart, and no table grows. The
  sign-in refill's (`SignInGrantOf`) is kept for good in planet's `charge_grants` (`lasting`). A caller with no account
  is paid nothing and planet is not asked. A failed payment loses the box or the answer, as before.
- **The lean.** `rpc_planet_shares` asks `GetTerritories` every 10 s and keeps each flag's share of the map
  (`territory.tiles / tiles`); a question is drawn under the registry's lock, so the draw only reads what was kept. Until
  planet first answers, every share is 0 and the draw is flat; a failed ask keeps the last shares.

## Its rule sets

**The offers keep rule sets of their own**, as the antibot does ([The antibot's rules](antibot-rules.md)), named like
planet's and played by name by the gamemaster: `default` holds every value, another set only what it changes, and plays
default's for the rest (`cpproto.Overlay`). `bonuses.v1.Rules`:

- `boxes.minInterval`, `maxInterval`, `maxChargesPerHour` — a box's wait and the hourly cap. Half the shorter of this
  `minInterval` and the quiz's is how far a box and a quiz keep apart ([Quizzes](quizzes.md))
- `kinds.*` — what a box or a quiz is drawn as: a weight per kind, `refill`, `spreadClicks`, `encloseClicks`, `bomb`,
  `shields`. 0 is never offered, nor a kind planet says a hand has no room for
- `offerTtl` — how long a box flies; it must outlast the flight the page draws
- `missRetry` — a box missed brings the next one this soon, once
- `activeWithin` — no box and no quiz for a caller that has not clicked for this long
- `spread.maxPerBox`, `enclose.maxPerBox`, `shield.minPerBox`, `shield.maxPerBox` — what a box or a right answer
  grants at most (and at least). The stacks those fill are planet's `charges.*`
- `welcome.within`, `welcome.boxAfter` — a newcomer's first box and first quiz ([A newcomer's first box and first
  quiz](welcome-bonus.md))
- `quiz.enabled`, `minInterval`, `maxInterval`, `maxChargesPerHour`, `offerTtl`, `answerWindow`, `missRetry`,
  `leaderBias` — the quizzes
- `signInRefill` — a guest's first sign-in earns a refill ([A guest's first sign-in earns a refill](sign-in-refill.md))
- `frozen` — the offers take no play (below)

A set that cannot play is refused with every reason, each named by its path in `bonuses.v1.Rules`
(`boxes.minInterval is 0s, and must be above 0`), in the words planet used for them. A set's own value is checked
against default's (a shortest wait over default's longest is refused), and a save of `default` is refused while another
set could not play over it. The set in force is one atomic pointer each reader reads at each use; a set taken applies
from the next sweep, as in planet ([What changes for what is already there](rules.md#what-changes-for-what-is-already-there)).
Each set taken is logged at Warn (`offer rules in force`), each save too (`admin offer rule set saved` or `refused`).
**Promo cites the offers' values as `bonuses.<path>`**, the place they held in planet's rules ([Promo](promo.md)).

- **`frozen` offers nothing and refuses every claim, opening and answer.** A slot that comes due while frozen is lost
  as one with nothing to offer (the next wait is drawn) and planet is asked no room. Until 2026-11-11 a frozen planet
  still offered boxes on their timer and refused only the claims.
- **The schedule plays the offers with planet**: the gamemaster calls `bonuses.v1.InternalService/PlayRuleSet` after
  planet's, so a battle's boxes come with its map; the antibot alone may go first (see
  [The gamemaster](gamemaster.md)). An entry's set must be kept by planet, the antibot and the offers.
- **No `bomb.off` of its own**: planet's room says it (above). A set weighing the bomb alone is no longer refused while
  bombs are off; each of its slots finds no room and is lost.

### Where the sets came from

Production held six planet sets, each with an offers' half under `bonuses`: `default`, `bridge`, `capital-siege`,
`frozen`, `unlimited-enclose` and `urf`. **Migration `20261111200000_from_planet` copies each one's offers' half and its
`frozen`** from `planet.rule_sets` into `bonuses.rule_sets`, under the same name, so the `frozen` set is frozen here
too; on a database with no planet schema it seeds production's default instead. Planet's migration of the same release
keeps `bonuses` in its sets (it reads it no more) and adds `charges`, so the copy reads it whichever module migrates
first; a later release strips it. `migrations_test.go` runs production's six sets of 2026-11-11 through the copy and
checks each comes across value for value and plays over default; the `urf` set keeps boxes every 30 s to 1 min.

## The freeze

**A claim, an opening and an answer are refused before the offer is spent** (`bonusesv1controller.NewFreezeInterceptor`),
with `FailedPrecondition` and a `FrozenRefusal` detail, while:

- the offers play a frozen set, or
- planet's `GetFreeze` answers frozen: its rules in force freeze the map, or an arena's freeze holds it
  ([Arena battles](arenas.md)) — one loopback call per claim, opening and answer, since an arena's freeze lasts seconds
  and is known only to planet.

These are the refusals planet applied to the three calls before the move (its freeze interceptor); the session check
moved with them, and the VPN blocklist never applied to them. The box or the quiz lapses as it would have.

## Events and metrics

- **`BonusClaimed`** after a caught box's charge is kept, **`QuizAnswered`** for each answer, right or wrong (a right
  one after its charge is kept), each for a caller with an account; heard by [missions](missions.md) and
  [progression](progression.md). Not in an outbox: a loss is a stat short by one.
- **The box facts** `BoxCaught`, `BoxLapsed`, `ForeignBoxClaimed`, through the module's own facts log and its 100 ms
  relay: each at a later time than the one before, so the [antibot](antibot.md)'s marks know one told again. The log's
  first time is past planet's last, since it reads the same wall clock after a restart.
- **The groups kept their names when the events moved**, and read what planet's streams still held first:
  `cpredis.SubscribeMoved` (see [Telling other modules what happened](composite-layer.md#telling-other-modules-what-happened)).
- **Every metric kept its name**: `bonus_claims_total`, `bonus_offers_total`, `quiz_answers_total` and the rest are
  registered by the same decorators. New: `bonus_session_checks`, the session verdicts of the three calls, which
  `click_session_checks` no longer counts.

## The sign-in refill

`player_created_subscriber` hears `player.v1.PlayerCreated` under group **`planet-charges-players`**, planet's name
kept, so it goes on where planet left the stream, and pays a refill through `GrantCharges` with `lasting`, once per
account (`SignInGrantOf`, the same id as before, so a grant planet made before the move is not made again). See
[A guest's first sign-in earns a refill](sign-in-refill.md).

## Tests

The registry's own (pace, apart, welcome, quizzes, the room's sweep, the frozen set, a click heard late), the use cases',
the handlers', the stream's and its sink's, `planetwords`' (`TestTheOffersAndTheWalletNameTheSameKinds`), the wallet's
and the shares' adapters, the subscribers' (`TestOnlyATakeAStrikeOrAnUnchangedTileIsHeard`), the rules package's, the
stores' (postgres), the migration's, and `TestTheQuizzesAreTheOffersAlone`, which lets no package of the module but the
offers and `module.go` import the bank.
