# The bonuses (`internal/bonuses/`)

**The bonuses module grants what a box caught, a right answer or a guest's first sign-in is worth.** It holds no charge:
the charges a player holds and spends are planet's wallet ([Charges](charges.md)). The rule of the cut: **granting is the
bonuses module's, holding and spending is planet's.** The module left planet on 2026-11-11 (step B of [Offers and the
wallet](offers-and-wallet.md)) as "the offers". Since, three cuts leave it with the reward alone: **the quizzes left it on
2026-11-13** ([The quizzes](quizzes-module.md)) and **the boxes on 2026-11-15** ([The boxes](boxes-module.md)). The last
cut, the player level, makes it hear progression.

It is downstream of planet like [missions](missions.md): it asks planet's wallet, and planet knows nothing of it. **It
speaks planet's, player's, the boxes' and the quizzes' language alone**: `cmd/api/modules_test.go` pins it
(`TestTheBonusesSpeakPlanetsPlayersTheBoxesAndTheQuizzesLanguageAlone`, which also pins that planet names no `bonuses`).

## What it holds

| package | what |
|---|---|
| `internal/bonuses` | a kind and its draw (`kind.go`), a box caught (`box.go`) and a right answer (`answer.go`), the ports it asks planet through (`wallet.go`), the sign-in grant (`sign_in.go`), its rules (`rules.go`, `message.go`) and the use cases `pay_caught_box`, `pay_right_answer` and `grant_sign_in_refill` |
| `internal/rules` | the list of its parts (`parts.All()`) and its fixtures (`rules.Seeded()`); the machinery is `cprules` ([Shared](shared.md)) |
| `internal/planetwords` | planet's words for a kind of charge (`planet.v1.BonusKind`) |
| `internal/subscribers` | the boxes caught heard from the boxes, the answers heard from the quizzes, the new players heard from player |
| `internal/bonusesv1controller` | the handlers |

Its postgres schema is `bonuses` (`bonuses.rule_sets`, `bonuses.rule_set_choice`). Its config is the `bonuses:` block:
`database` and `redis`; `offers.forgetAfter` and `offers.sweepInterval` are the boxes' since 2026-11-15.

## The wire (`proto/bonuses/v1/`)

- **`BonusService`**, public, routed by Caddy (`/bonuses.v1.BonusService/*`): `GetGrantRules`, what the page shows of the
  bonuses' rules, `sign_in_refill` today. A GET no cache keeps; the page reads it once a load. **It has no stream since
  2026-11-15**: the boxes' went to `boxes.v1.BoxService`, and a `rules_changed` alone was not worth a stream a page (the
  owner's call). Only `default` sets `signInRefill` in production, so a scheduled set never moves it; the operator's edit
  of it reaches an open page at its next load, and every deploy reloads the pages. `ClaimBonus` is the boxes' `ClaimBox`
  since 2026-11-15, `OpenQuiz` and `AnswerQuiz` the quizzes' since 2026-11-13.
- **Events**: none. `BonusClaimed` and the box facts are the boxes' since 2026-11-15 (`boxes.v1.BoxClaimed`, `BoxCaught`,
  `BoxLapsed`, `ForeignBoxClaimed`), `QuizAnswered` the quizzes' since 2026-11-13.
- **`InternalService`**: `PlayRuleSet` (the gamemaster's; the latest `decided_at` wins; `FailedPrecondition` for a set
  it keeps none of or cannot play), `GetRuleSetNames` and `GetRuleSet` (default's values with the set's own over them;
  `NotFound`), for the gamemaster and promo.
- **`AdminService`**: `GetRules` and `SaveRuleSet`, which the backoffice's Box & quiz rules page calls and Claude calls as
  `bonuses_get_rules` and `bonuses_save_rule_set` ([Claude as a connector](connector.md)).

## What it asks of planet, in planet's words

| port | planet's answer | when |
|---|---|---|
| `Room.Full` | `InternalService/GetRoom` | once per box caught and per right answer |
| `Wallet.Pay` | `InternalService/GrantCharges` | each box caught, right answer, sign-in refill |

- **Room.** Planet answers, for each account asked, the kinds its hand has no room for: a full stack, a charge held
  where one is the most, and **a bomb while planet's `charges.bomb.off` is on**, or an enclosure while its
  `charges.enclose.off` is on. So `bomb.off` and `enclose.off` are planet's rules alone: no hand has room for one then.
  An error is tried again by the bus.
- **Pay.** Planet grants once per grant id. A box's or an answer's id is its token's (`OfferGrantOf`) and planet keeps it
  10 minutes in memory (`lasting` unset): a retry never outlives a restart, and no table grows. The sign-in refill's
  (`SignInGrantOf`) is kept for good in planet's `charge_grants` (`lasting`). A caller with no account is paid nothing
  and planet is not asked. The pay answers no hand: the page reads the grant on planet's stream.

## A box caught

**The bonuses module pays each box the boxes tell caught** (`boxes.v1.BoxClaimed`, group `bonuses-boxes`,
`pay_caught_box_usecase`). It reads only that stream: a box the bonuses module offered before 2026-11-15 was paid as it
was caught.

- **What the box holds is drawn at the catch** (`bonuses.CaughtBox.Reward`), as it was drawn at the offer before: a kind
  from `kinds`, among those planet's room says the account's hand has room for (`GetRoom`, one account), and its amount
  from `spread`, `enclose` and `shield`. **A newcomer's early box pays a spread or nothing** (`BoxClaimed.welcome`):
  never a refill or a bomb, as before.
- **A box may pay nothing**, which it could not before: the boxes know no kind and ask no room, so a box comes to a full
  hand, or to one with room only for kinds weighed 0, as a quiz is asked of one. Counted in
  `bonus_box_rewards_total{outcome}` (`paid`, `nothing`, `failed`), by `prom_pay_caught_box`.
- **The grant id is the box's token's** (`OfferGrantOf`, as before the cut): the bus may tell a catch twice, and planet
  pays it once. A redraw on a second telling changes nothing.
- **The reward reaches the page on planet's stream** (`charges_granted`, with what it added): `ClaimBox` answers before
  it, about a second earlier (the boxes' outbox sends every second).

## A right answer

**The bonuses module pays the right answers the quizzes tell** (`quizzes.v1.QuizAnswered`, group `bonuses-right-answers`,
`pay_right_answer_usecase`), the same way: a kind drawn from `kinds` among those the hand has room for
(`bonuses.Answer.Reward`), **shields or nothing for a newcomer's first quiz**, under the quiz's token. A wrong answer pays
nothing.

## Its rule sets

**The bonuses module keeps rule sets of its own**, named like planet's and played by name by the gamemaster: `default`
holds every value, another set only what it changes, and plays default's for the rest (`cpproto.Overlay`).
`bonuses.v1.Rules`:

- `kinds.*` — what a box or a right answer is drawn as: a weight per kind, `refill`, `spreadClicks`, `encloseClicks`,
  `bomb`, `shields`. 0 is never drawn, nor a kind planet says a hand has no room for
- `spread.maxPerBox`, `enclose.maxPerBox`, `shield.minPerBox`, `shield.maxPerBox` — what a box or a right answer
  grants at most (and at least). The stacks those fill are planet's `charges.*`
- `signInRefill` — a guest's first sign-in earns a refill ([A guest's first sign-in earns a refill](sign-in-refill.md))
- `quiz` (10) — `reserved` since 2026-11-13: the quizzes' rules are `quizzes.v1.Rules`
- `boxes`, `offerTtl`, `missRetry`, `activeWithin`, `welcome`, `frozen` (1, 3, 4, 5, 9, 12) — `reserved` since
  2026-11-15: the boxes' rules are `boxes.v1.Rules`, copied from here ([The boxes](boxes-module.md#its-rule-sets)). A set
  kept from before still holds them, read no more. **The bonuses module has no `frozen` since**: the boxes and the
  quizzes refuse a claim and an answer, and what they accepted is still paid

A set that cannot play is refused with every reason, each named by its path in `bonuses.v1.Rules`. The set in force is one
atomic pointer each reader reads at each use. Each set taken is logged at Warn (`offer rules in force`), each save too
(`admin offer rule set saved` or `refused`). **Promo cites these values as `bonuses.<path>`**, the place they held in
planet's rules ([Promo](promo.md)).

- **The schedule plays the bonuses' set with planet**: the gamemaster calls `bonuses.v1.InternalService/PlayRuleSet`
  after planet's and the boxes', then the quizzes' (see [The gamemaster](gamemaster.md)). An entry's set must be kept by
  planet, the antibot, the boxes, the bonuses module and the quizzes. The gamemaster plays the bonuses' set until the
  player level, when the reward hears progression instead.

### Where the sets came from

Production held six planet sets, each with an offers' half under `bonuses`: `default`, `bridge`, `capital-siege`,
`frozen`, `unlimited-enclose` and `urf`. **Migration `20261111200000_from_planet` copies each one's offers' half and its
`frozen`** from `planet.rule_sets` into `bonuses.rule_sets`, under the same name; on a database with no planet schema it
seeds production's default instead. The quizzes' migration of 2026-11-13 and the boxes' of 2026-11-15 copy their halves
out of these sets in turn; the halves stay here, read no more, until a later release strips them.

## A forgotten account

Nothing the bonuses module keeps names an account for longer than a process lives.

## The sign-in refill

`player_created_subscriber` hears `player.v1.PlayerCreated` under group **`planet-charges-players`**, planet's name
kept, so it goes on where planet left the stream, and pays a refill through `GrantCharges` with `lasting`, once per
account (`SignInGrantOf`, the same id as before, so a grant planet made before the move is not made again). See
[A guest's first sign-in earns a refill](sign-in-refill.md).

## Tests

The kinds' draw and amounts, a box's and an answer's welcome kind, the use cases' (a box's and a right answer's reward, the
welcome's spread and shields, a hand with no room, a box that pays nothing), the box's metric, `planetwords'`
(`TestTheOffersAndTheWalletNameTheSameKinds`), the wallet's adapter, the subscribers', and the rules package's (its values,
a set kept with the boxes' half, and `cprules`' contract suites on its migrations).
