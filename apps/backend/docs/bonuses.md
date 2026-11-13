# The offers (`internal/bonuses/`)

**The bonuses module offers the boxes, and grants what a caught box, a right answer or a guest's first sign-in is
worth.** It holds no charge: the charges a player holds and spends are planet's wallet ([Charges](charges.md)). The rule
of the cut: **granting is the offers', holding and spending is planet's.** The module left planet on 2026-11-11 (step B
of [Offers and the wallet](offers-and-wallet.md)). **The quizzes left it on 2026-11-13** for a module of their own
([The quizzes](quizzes-module.md)), the first of three cuts that leave it with the reward alone: the boxes go next.

It is downstream of planet like [missions](missions.md): it hears planet's clicks, asks planet's wallet, and planet
knows nothing of it. **It speaks planet's, player's and the quizzes' language alone**: `cmd/api/modules_test.go` pins it
(`TestTheBonusesSpeakPlanetsPlayersAndTheQuizzesLanguageAlone`, which also pins that planet names no `bonuses`).

## What it holds

| package | what |
|---|---|
| `internal/bonuses` | the registry (`bonus.go`, `welcome.go`, `entrant.go`), the ports it asks planet through (`wallet.go`), a right answer's reward (`answer.go`), the box facts (`box_facts`), its rules (`rules.go`, `message.go`) and the use cases `claim_bonus`, `pay_right_answer`, `grant_sign_in_refill` and `listen_for_events` |
| `internal/rules` | the list of its parts (`parts.All()`) and its fixtures (`rules.Seeded()`); the machinery of its rule sets (snapshots, sets, the set in force, the choice, their postgres stores, the play and save use cases) is `cprules`, shared with planet and the antibot ([Shared](shared.md)) |
| `internal/facts` | its own facts log and relay, for the box facts |
| `internal/planetwords` | planet's words for a kind of charge and a hand (`planet.v1.BonusKind`, `ChargesHeld`) |
| `internal/subscribers` | the clicks heard from planet, the new players heard from player, the answers heard from the quizzes |
| `internal/bonusesv1controller` | the handlers, the session interceptor and reader, the freeze interceptor |

Its postgres schema is `bonuses` (`bonuses.rule_sets`, `bonuses.rule_set_choice`; the quiz answers and ratings moved to
schema `quizzes` on 2026-11-13); nothing of a box or of an offer in flight is written there, as nothing was in planet's.
Its config is the `bonuses:` block: `database`, `redis`, and `offers.forgetAfter` (5 min, a caller with no stream open is
forgotten) and `offers.sweepInterval` (1 s), which were planet's top-level `bonus:` block.

## The wire (`proto/bonuses/v1/`)

- **`BonusService`**, public, routed by Caddy (`/bonuses.v1.BonusService/*`):
  - `ClaimBonus`: the request and answer planet's was, in planet's words for what it grants (`planet.v1.BonusKind`,
    `planet.v1.ChargesHeld`). `NotFound` for a box not offered to the caller, gone or taken; `FailedPrecondition` with
    `bonuses.v1.FrozenRefusal` while frozen (below). The click token is checked as planet checked it
    (`cpconnect.Attested`, counted in `bonus_session_checks`). `OpenQuiz` and `AnswerQuiz` are the quizzes' since
    2026-11-13.
  - `ListenForEvents`: a stream of its own, `BonusEvent` — `heartbeat`, `bonus_offered` (to the caller alone),
    `bonus_taken` (to every stream) and `rules_changed` (the offers took another set); `quiz_offered` (4) and
    `bonus_taken.quiz_subject_country_id` (3) are `reserved`, the quizzes' stream's now. A heartbeat at once and every `httpServer.streamHeartbeat`, a clean end on the drain
    ([The live streams](live-streams.md)). Its session reader refuses nothing: a signed-in account's stream attends as
    that account, any other as its address, which every guest behind it shares (`bonuses.EntrantOf`). **An offer goes
    only to a caller with this stream open** (`caller.watching()`).
  - `GetGrantRules`: what the page shows of the offers' rules, `sign_in_refill` today. A GET no cache keeps;
    `rules_changed` sends the page to read it again.
- **Events** (`events.proto`): `BonusClaimed` and the box facts `BoxCaught`, `BoxLapsed`, `ForeignBoxClaimed`, with
  planet's fields and numbers, so a payload planet told reads as the new type. `QuizAnswered` is the quizzes' since
  2026-11-13.
- **`InternalService`**: `PlayRuleSet` (the gamemaster's; the latest `decided_at` wins; `FailedPrecondition` for a set
  the offers keep none of or cannot play), `GetRuleSetNames` and `GetRuleSet` (default's values with the set's own over
  them; `NotFound`), for the gamemaster and promo.
- **`AdminService`**: `GetRules` and `SaveRuleSet`, which the backoffice's Box & quiz rules page calls and Claude calls as
  `bonuses_get_rules` and `bonuses_save_rule_set` ([Claude as a connector](connector.md)). `ListQuizQuestions` is the
  quizzes' `quizzes_list_quiz_questions` since 2026-11-13.

## What it asks of planet, in planet's words

| port | planet's answer | when |
|---|---|---|
| activity (`Registry.Clicked`) | `planet.v1.ClickTried`, group `bonuses-clicks` | every click planet tried |
| `Room.Full` | `InternalService/GetRoom` | once per sweep for each caller due, outside the registry's lock; once per right answer |
| `Wallet.Pay` | `InternalService/GrantCharges` | each box caught, right answer, sign-in refill |
| the map's freeze | `InternalService/GetFreeze` | each claim |

- **Activity.** `click_tried_subscriber` keeps the outcomes `TAKEN`, `STRUCK` and `UNCHANGED`, exactly what planet's
  `bonus_click` heard inside the click chain before the move (see [What `Clicked` hears](offers-and-wallet.md#what-clicked-hears)).
  A click carries its caller (scope, account, linked), `tried_at` and the account's age, read off its UUIDv7 id
  (`cpsession.AccountID.CreatedAt`). `dry` is the quizzes' to read now.
  - **The delay.** Planet's `facts.Relay` sends what its log took every 100 ms in one pipelined `XADD`, and the group's
    blocking read returns as soon as an entry lands: a click is heard about 0.1 s after it was tried, a busy 100 ms in
    one batch, in order. The registry times a click by `tried_at`, so `activeWithin` (2 min) counts from the try as
    before; a click heard after `activeWithin` (a backlog read at a boot) is no sign of play and is dropped.
  - A `ClickTried` Redis refused (counted in `events_unpublished_total`) is a click the offers never hear, as the
    antibot never does.
- **Room.** Planet answers, for each account asked, the kinds its hand has no room for: a full stack, a charge held
  where one is the most, and **a bomb while planet's `charges.bomb.off` is on**, or an enclosure while its
  `charges.enclose.off` is on. So `bomb.off` and `enclose.off` are planet's rules alone: the offers never offer a bomb or
  an enclosure then, since no hand has room for one. An account with room for everything is left out of
  the answer. An error loses the slot, as before ([Room](offers-and-wallet.md#room)).
- **Pay.** Planet grants once per grant id and answers what the stack kept and the hand after. A box's or an answer's id
  is its token's (`OfferGrantOf`) and planet keeps it 10 minutes in memory (`lasting` unset), since the offer itself
  lives only in memory and both modules share one process: a retry never outlives a restart, and no table grows. The
  sign-in refill's (`SignInGrantOf`) is kept for good in planet's `charge_grants` (`lasting`). A caller with no account
  is paid nothing and planet is not asked. A failed payment loses the box, as before.

## A right answer

**The bonuses module pays the right answers the quizzes tell** (`quizzes.v1.QuizAnswered`, group `bonuses-right-answers`,
`pay_right_answer_usecase`). It reads only that stream: an answer the offers told before 2026-11-13 was paid as it was
answered. A wrong answer pays nothing.

- **The reward is a box's, drawn at the answer** (`bonuses.Answer.Reward`): a kind from `kinds`, among those planet's
  room says the account's hand has room for (`GetRoom`, one account), and its amount from `spread`, `enclose` and
  `shield`. **A newcomer's first quiz pays shields or nothing** (`QuizAnswered.welcome`): never a refill or a bomb, as
  before. A hand with room for nothing is paid nothing: the quizzes, which know no kind, no longer skip a quiz for it.
- **The grant id is the quiz's token's** (`OfferGrantOf`, as before the cut): the bus may tell an answer twice, and planet
  pays it once. A redraw on a second telling changes nothing, since planet keeps no second grant of the id.
- **The reward reaches the page on planet's stream** (`charges_granted`, with what it added): `AnswerQuiz` answers before
  it. See [Charges](charges.md#a-grant-on-the-stream).
- **The hourly cap is the quizzes'** (`maxRightAnswersPerHour`); `boxes.maxChargesPerHour` counts boxes alone, as before.

## Its rule sets

**The offers keep rule sets of their own**, as the antibot does ([The antibot's rules](antibot-rules.md)), named like
planet's and played by name by the gamemaster: `default` holds every value, another set only what it changes, and plays
default's for the rest (`cpproto.Overlay`). `bonuses.v1.Rules`:

- `boxes.minInterval`, `maxInterval`, `maxChargesPerHour` — a box's wait and the hourly cap. A box and a quiz no longer
  keep apart (dropped 2026-11-13, [The quizzes](quizzes-module.md))
- `kinds.*` — what a box or a right answer is drawn as: a weight per kind, `refill`, `spreadClicks`, `encloseClicks`,
  `bomb`, `shields`. 0 is never drawn, nor a kind planet says a hand has no room for
- `offerTtl` — how long a box flies; it must outlast the flight the page draws
- `missRetry` — a box missed brings the next one this soon, once
- `activeWithin` — no box for a caller that has not clicked for this long
- `spread.maxPerBox`, `enclose.maxPerBox`, `shield.minPerBox`, `shield.maxPerBox` — what a box or a right answer
  grants at most (and at least). The stacks those fill are planet's `charges.*`
- `welcome.within`, `welcome.boxAfter` — a newcomer's first box ([A newcomer's first box and first
  quiz](welcome-bonus.md))
- `quiz.*` — `reserved` since 2026-11-13: the quizzes' rules are `quizzes.v1.Rules`, copied from here
  ([The quizzes](quizzes-module.md#its-rule-sets)). A set kept from before still holds its `quiz` block, read no more
- `signInRefill` — a guest's first sign-in earns a refill ([A guest's first sign-in earns a refill](sign-in-refill.md))
- `frozen` — no box is offered and no claim taken (below); a right answer the quizzes accepted is still paid

A set that cannot play is refused with every reason, each named by its path in `bonuses.v1.Rules`
(`boxes.minInterval is 0s, and must be above 0`), in the words planet used for them. A set's own value is checked
against default's (a shortest wait over default's longest is refused), and a save of `default` is refused while another
set could not play over it. The set in force is one atomic pointer each reader reads at each use; a set taken applies
from the next sweep, as in planet ([What changes for what is already there](rules.md#what-changes-for-what-is-already-there)).
Each set taken is logged at Warn (`offer rules in force`), each save too (`admin offer rule set saved` or `refused`).
**Promo cites the offers' values as `bonuses.<path>`**, the place they held in planet's rules ([Promo](promo.md)).

- **`frozen` offers nothing and refuses every claim.** A slot that comes due while frozen is lost as one with nothing to
  offer (the next wait is drawn) and planet is asked no room. Until 2026-11-11 a frozen planet still offered boxes on
  their timer and refused only the claims. The quizzes have a `frozen` of their own.
- **The schedule plays the offers with planet**: the gamemaster calls `bonuses.v1.InternalService/PlayRuleSet` after
  planet's, then the quizzes', so a battle's boxes and quizzes come with its map; the antibot alone may go first (see
  [The gamemaster](gamemaster.md)). An entry's set must be kept by planet, the antibot, the offers and the quizzes.
- **No `bomb.off` of its own**: planet's room says it (above). A set weighing the bomb alone is no longer refused while
  bombs are off; each of its slots finds no room and is lost.

### Where the sets came from

Production held six planet sets, each with an offers' half under `bonuses`: `default`, `bridge`, `capital-siege`,
`frozen`, `unlimited-enclose` and `urf`. **Migration `20261111200000_from_planet` copies each one's offers' half and its
`frozen`** from `planet.rule_sets` into `bonuses.rule_sets`, under the same name, so the `frozen` set is frozen here
too; on a database with no planet schema it seeds production's default instead. Planet's migration of the same release
keeps `bonuses` in its sets (it reads it no more) and adds `charges`, so the copy reads it whichever module migrates
first; a later release strips it.

## The freeze

**A claim is refused before the offer is spent** (`bonusesv1controller.NewFreezeInterceptor`),
with `FailedPrecondition` and a `FrozenRefusal` detail, while:

- the offers play a frozen set, or
- planet's `GetFreeze` answers frozen: its rules in force freeze the map, or an arena's freeze holds it
  ([Arena battles](arenas.md)) — one loopback call per claim, since an arena's freeze lasts seconds and is known only to
  planet.

These are the refusals planet applied to the claim before the move (its freeze interceptor); the session check moved
with it, and the VPN blocklist never applied to it. The box lapses as it would have. The quizzes refuse their opening
and answer the same way.

## Events and metrics

- **`BonusClaimed`** after a caught box's charge is kept, for a caller with an account; heard by
  [missions](missions.md) and [progression](progression.md). Not in an outbox: a loss is a stat short by one.
- **The box facts** `BoxCaught`, `BoxLapsed`, `ForeignBoxClaimed`, through the module's own facts log and its 100 ms
  relay: each at a later time than the one before, so the [antibot](antibot.md)'s marks know one told again. The log's
  first time is past planet's last, since it reads the same wall clock after a restart.
- **The groups kept their names when the events moved**, and read what planet's streams still held first:
  `cpredis.SubscribeMoved` (see [Telling other modules what happened](composite-layer.md#telling-other-modules-what-happened)).
- **Every metric kept its name**: `bonus_claims_total`, `bonus_offers_total` and the rest are registered by the same
  decorators; the quizzes' moved with them. `bonus_session_checks` counts the claims' session verdicts, which
  `click_session_checks` no longer counts.

## A forgotten account

Nothing the offers keep names an account for longer than a process lives: the answers it gave, and the group that blanks
them (`bonuses-answers-forgotten`), are the quizzes' since 2026-11-13.

## The sign-in refill

`player_created_subscriber` hears `player.v1.PlayerCreated` under group **`planet-charges-players`**, planet's name
kept, so it goes on where planet left the stream, and pays a refill through `GrantCharges` with `lasting`, once per
account (`SignInGrantOf`, the same id as before, so a grant planet made before the move is not made again). See
[A guest's first sign-in earns a refill](sign-in-refill.md).

## Tests

The registry's own (pace, welcome, the room's sweep, the frozen set, a click heard late), the use cases' (a right
answer's reward, the welcome's shields, a hand with no room), the handlers', the stream's and its sink's, `planetwords`'
(`TestTheOffersAndTheWalletNameTheSameKinds`), the wallet's adapter, the subscribers'
(`TestOnlyATakeAStrikeOrAnUnchangedTileIsHeard`, a wrong answer paid nothing), and the rules package's (its values, and
`cprules`' contract suites on its migrations).
