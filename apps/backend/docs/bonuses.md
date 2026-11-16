# The bonuses (`internal/bonuses/`)

**The bonuses module grants what a box caught, a right answer, a level reached or a guest's first sign-in is worth.** It holds no charge:
the charges a player holds and spends are planet's wallet ([Charges](charges.md)). The rule of the cut: **granting is the
bonuses module's, holding and spending is planet's.** The module left planet on 2026-11-11 (step B of [Offers and the
wallet](offers-and-wallet.md)) as "the offers". Since, three cuts leave it with the reward alone: **the quizzes left it on
2026-11-13** ([The quizzes](quizzes-module.md)) and **the boxes on 2026-11-15** ([The boxes](boxes-module.md)). The last
cut, the player level, does not make it hear progression: progression tells it what each account unlocked, in its words
(see [The gates](#the-gates)).

It is downstream of planet like [missions](missions.md): it asks planet's wallet, and planet knows nothing of it. **It
speaks planet's, player's, the boxes' and the quizzes' language alone**: `cmd/api/modules_test.go` pins it
(`TestTheBonusesSpeakPlanetsPlayersTheBoxesAndTheQuizzesLanguageAlone`, which also pins that planet names no `bonuses`).
**It knows no level.** The gamemaster plays its rule sets and progression hears the gamemaster, so it may not hear
progression: progression sends it a command in its words instead (`TestProgressionTellsTheBonusesWhatEachAccountUnlocked`).

## What it holds

| package | what |
|---|---|
| `internal/bonuses` | a kind and its draw (`kind.go`), a box caught (`box.go`), a right answer (`answer.go`) and a level reached (`level_up.go`), the ports it asks planet through (`wallet.go`), the kinds an account unlocked (`unlocked.go`), the sign-in grant (`sign_in.go`), its rules and gates (`rules.go`, `message.go`) and the use cases `pay_caught_box`, `pay_right_answer`, `pay_level_up` and `grant_sign_in_refill` |
| `internal/unlocks` | what progression last decided an account may be drawn (`Unlocks`, the latest by its time), its stores and the use cases `keep_unlocks` and `forget_unlocks` |
| `internal/commands` | `bonuses.v1.KeepUnlocks` and `PayLevelUp`, handled |
| `internal/rules` | the list of its parts (`parts.All()`) and its fixtures (`rules.Seeded()`); the machinery is `cprules` ([Shared](shared.md)) |
| `internal/planetwords` | planet's words for a kind of charge (`planet.v1.BonusKind`) |
| `internal/subscribers` | the boxes caught heard from the boxes, the answers heard from the quizzes, the new and the forgotten players heard from player |
| `internal/bonusesv1controller` | the handlers |

Its postgres schema is `bonuses` (`bonuses.rule_sets`, `bonuses.rule_set_choice`, `bonuses.unlocks`). Its config is the `bonuses:` block:
`database` and `redis`; `offers.forgetAfter` and `offers.sweepInterval` are the boxes' since 2026-11-15.

## The wire (`proto/bonuses/v1/`)

- **`BonusService`**, public, routed by Caddy (`/bonuses.v1.BonusService/*`): `GetGrantRules`, what the page shows of the
  bonuses' rules: `sign_in_refill`, `gates_on`, and `open_kinds` in planet's words (`planet.v1.BonusKind`). The page reads
  the level each kind unlocks at from progression (`GetLevelResponse.unlocks`) and draws a kind locked while
  `gates_on`, not in `open_kinds` and above the player's level. A battle's set changes `gates_on`, and the page reads
  `GetGrantRules` again when the set in force changes. A GET no cache keeps; the page reads it once a load. **It has no stream since
  2026-11-15**: the boxes' went to `boxes.v1.BoxService`, and a `rules_changed` alone was not worth a stream a page (the
  owner's call). Only `default` sets `signInRefill` in production, so a scheduled set never moves it; the operator's edit
  of it reaches an open page at its next load, and every deploy reloads the pages. `ClaimBonus` is the boxes' `ClaimBox`
  since 2026-11-15, `OpenQuiz` and `AnswerQuiz` the quizzes' since 2026-11-13.
- **Commands**: `bonuses.v1.KeepUnlocks{account_id, kinds, decided_at}` and `PayLevelUp{grant_id, account_id, level,
  unlocked}` (`commands.proto`), which progression sends through its outbox, each handled on its own stream
  (`commands:bonuses.v1.…`, see [The gates](#the-gates) and [A level reached](#a-level-reached)).
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
| `Wallet.Pay` | `InternalService/GrantCharges` | each box caught, right answer, level reached, sign-in refill |

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
  from `kinds`, among those planet's room says the account's hand has room for (`GetRoom`, one account) and, while the
  gates are on, those the account may be drawn ([The gates](#the-gates)), and its amount from `spread`, `enclose` and
  `shield`. **A newcomer's early box pays a spread or nothing** (`BoxClaimed.welcome`):
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
`pay_right_answer_usecase`), the same way: a kind drawn from `kinds` among those the hand has room for and the gates
let (`bonuses.Answer.Reward`), **shields or nothing for a newcomer's first quiz**, under the quiz's token. A wrong answer pays
nothing.

## A level reached

**The bonuses module pays each level-up progression sends** (`bonuses.v1.PayLevelUp`, `commands:bonuses.v1.PayLevelUp`,
`pay_level_up_usecase`), as a box caught: one kind drawn from `kinds` among those the hand has room for and the gates
let (`bonuses.LevelUp.Reward`), in a box's amounts. Progression sends one per raise, however many levels it jumps, from
level 2 on, and none for a reconciliation ([Levels](levels.md#the-level-up-bonus)).

- **The kinds the level unlocks come with the command** (`unlocked`): the draw does not read the unlocks kept, which
  `KeepUnlocks` brings on another stream, in no order with this one.
- **The grant is progression's** (`grant_id`, one per account and level), paid with `lasting`: planet keeps it for good
  in `charge_grants`, so a command told again any day pays nothing more.
- A level-up whose kinds all weigh 0, or a full hand, pays nothing, as a box.

## The gates

**While a set's gates are on, a win is drawn only from the kinds open to everyone and the ones progression unlocked for
the account**; off, from every kind (`Rules.MayDraw`). The ladder is progression's: which level unlocks which kind, one
type per rung ([Levels](levels.md#the-unlocks)). The bonuses module knows no level, only the kinds it was told.

- **`gates.on`**, and **`gates.open.*`**, a switch per kind: the kinds anyone may be drawn while the gates are on, a
  guest too. The migrations seed `default` with `on: false`, refill, spread and shields open (`…_unlocks`; the virus
  closed, `…_virus_offers`, the mine, `…_mine_offers`, and the rally, `…_rally_offers`). The operator turned the gates
  on at the reveal; since spread and shields became rungs of the ladder (2026-11-16), refill alone is open in `default`,
  closed after a `ReconcileLevels` ([Levels](levels.md#the-unlocks)). A battle's set keeps them off: newcomers who come
  for a battle fight with every kind. That is data, not code.
- **A guest has nothing unlocked**: progression keeps no level for one, so it draws the open kinds alone.
- **Gates decide only what is drawn.** A charge held stays usable whatever the gates: planet checks no level.
- **A win whose kinds it may be drawn all weigh 0 pays nothing**, as a full hand does; a box's metric counts it
  `nothing`. **A set with the gates on whose open kinds all weigh 0 is refused**: a guest's box could never be anything.
- **A newcomer's welcome is its one kind or nothing**: a box's spread, a quiz's shields, each only if the account may be
  drawn it.

**The kinds unlocked are kept per account** (`unlocks.Unlocks`, `bonuses.unlocks(account_id, kinds unlocked_kind[],
decided_at)`): what progression decided last. `bonuses.v1.KeepUnlocks` names the whole list at a time, never one change;
`keep_unlocks_usecase` keeps it only when its `decided_at` is after the one kept, under an advisory lock per account, so a
command told twice, late or out of order changes nothing, and fewer kinds than before are fewer. Progression sends it with
each raise and for every account of `ReconcileLevels`, through its outbox. Each draw reads the account's row (a guest has
none). A kind planet's `BonusKind` names that the bonuses module never draws is refused, and buried after its retries.

A new kind on the ladder needs its value in `unlocked_kind` (a migration) and its switch in `gates.open`.

## Its rule sets

**The bonuses module keeps rule sets of its own**, named like planet's and played by name by the gamemaster: `default`
holds every value, another set only what it changes, and plays default's for the rest (`cpproto.Overlay`).
`bonuses.v1.Rules`:

- `kinds.*` — what a box or a right answer is drawn as: a weight per kind, `refill`, `spreadClicks`, `encloseClicks`,
  `bomb`, `shields`, `virus`, `mine`, `rally`. 0 is never drawn, nor a kind planet says a hand has no room for. `virus`,
  `mine` and `rally` weigh 0 in `default` until the operator weighs them ([The virus](virus.md), [The mine](mine.md),
  [The rally](rally.md))
- `spread.maxPerBox`, `enclose.maxPerBox`, `shield.minPerBox`, `shield.maxPerBox`, `mine.maxPerBox`, `rally.maxPerBox` —
  what a box or a right answer grants at most (and at least): a mine box holds 1 or 2 (`mine.maxPerBox` 2), a rally box
  one (`rally.maxPerBox` 1). The stacks those fill are planet's `charges.*`
- `signInRefill` — a guest's first sign-in earns a refill ([A guest's first sign-in earns a refill](sign-in-refill.md))
- `gates.on`, `gates.open.refill`, `spreadClicks`, `encloseClicks`, `bomb`, `shields`, `virus`, `mine`, `rally` — whether a win is drawn
  only from the open kinds and the ones the account unlocked ([The gates](#the-gates))
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
  planet, the antibot, the boxes, the bonuses module and the quizzes. So the bonuses module never hears progression:
  progression tells it what each account unlocked as a command ([The gates](#the-gates)).

### Where the sets came from

Production held six planet sets, each with an offers' half under `bonuses`: `default`, `bridge`, `capital-siege`,
`frozen`, `unlimited-enclose` and `urf`. **Migration `20261111200000_from_planet` copies each one's offers' half and its
`frozen`** from `planet.rule_sets` into `bonuses.rule_sets`, under the same name; on a database with no planet schema it
seeds production's default instead. The quizzes' migration of 2026-11-13 and the boxes' of 2026-11-15 copy their halves
out of these sets in turn; the halves stay here, read no more, until a later release strips them.

## A forgotten account

**The kinds unlocked go with the player**: `player.v1.PlayerForgotten` (group `bonuses-unlocks-forgotten`,
`forget_unlocks_usecase`) deletes the account's row. A `KeepUnlocks` progression recorded before the forget can write it
again, as a late fact does in progression. Nothing else the bonuses module keeps names an account for longer than a
process lives.

## The sign-in refill

`player_created_subscriber` hears `player.v1.PlayerCreated` under group **`planet-charges-players`**, planet's name
kept, so it goes on where planet left the stream, and pays a refill through `GrantCharges` with `lasting`, once per
account (`SignInGrantOf`, the same id as before, so a grant planet made before the move is not made again). See
[A guest's first sign-in earns a refill](sign-in-refill.md).

## Tests

The kinds' draw and amounts, a box's and an answer's welcome kind, a level-up's draw and grant, the gates (a kind not unlocked never drawn, a guest
drawn the open kinds, a set with the gates off giving a guest a bomb, a win whose kinds all weigh 0, the charges held
untouched), the use cases' (a box's and a right answer's reward, the welcome's spread and shields, a hand with no room, a
box that pays nothing), the unlocks (the latest decision kept, its stores' contract suite on postgres and in memory, the
command's handler, the forget), the box's metric, `planetwords'`
(`TestTheOffersAndTheWalletNameTheSameKinds`), the wallet's adapter, the subscribers', and the rules package's (its values,
a set kept with the boxes' half, and `cprules`' contract suites on its migrations).
