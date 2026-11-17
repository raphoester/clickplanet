# The boxes (`internal/boxes/`)

**The boxes module offers the boxes and pays nothing.** It owns the box schedule, the offer to one caller, the claim, a
newcomer's early box and the box facts. It tells each box caught (`boxes.v1.BoxClaimed`); what the box holds is drawn and
paid by the bonuses module, which hears it ([The bonuses](bonuses.md)). The module left `bonuses` on 2026-11-15, the
second of three cuts that leave bonuses with the reward alone, so that a reward can later depend on the player's level
(the owner's Claude Doc "Boxes and quizzes leave bonuses"). How a box is offered is in [Bonus boxes](bonus-boxes.md).

It is downstream of planet: it hears planet's clicks, and asks planet the freeze. **It asks no hand what it holds**
(below). **It speaks
planet's language alone**, and nothing it speaks to hears it back: `cmd/api/modules_test.go` pins it
(`TestTheBoxesSpeakPlanetsLanguageAlone`).

## What it holds

| package | what |
|---|---|
| `internal/boxes` | the registry (`box.go`, `welcome.go`, `entrant.go`, `caller.go`), the box facts (`box_facts`), its rules (`rules.go`, `message.go`) and the use cases `claim_box` and `listen_for_events`; moved from `bonuses/internal/bonuses` |
| `internal/facts` | its own facts log and relay, for the box facts; moved from `bonuses/internal/facts` |
| `internal/rules` | the list of its parts (`parts.All()`) and its fixtures (`rules.Seeded()`); the machinery is `cprules` ([Shared](shared.md)) |
| `internal/subscribers` | the clicks heard from planet |
| `internal/boxesv1controller` | the handlers, the session interceptor and reader, the freeze interceptor |

Its postgres schema is `boxes`: `boxes.rule_sets`, `boxes.rule_set_choice` and `boxes.outbox`. Nothing of a box in flight
is written there. Its config is the `boxes:` block: `database`, `redis`, and `offers.forgetAfter` (5 min) and
`offers.sweepInterval` (1 s), which were the bonuses module's. Production gives it one postgres connection, which
progression gave up (4 to 3), so `max_connections` stays 42.

## The wire (`proto/boxes/v1/`)

- **`BoxService`**, public, routed by Caddy (`/boxes.v1.BoxService/*`):
  - `ClaimBox`: a box offered to the caller, caught. **It answers nothing of what the box held**: the bonuses module draws
    it, and the reward comes on planet's stream ([Charges](charges.md#a-grant-on-the-stream)). `NotFound` for a box not
    offered to the caller, gone or caught; `FailedPrecondition` with `boxes.v1.FrozenRefusal` while frozen. The click
    token is checked (`cpconnect.Attested`, counted in `bonus_session_checks`, the name it had).
  - `ListenForEvents`: a stream of `BoxEvent`: `heartbeat` and `box_offered` (to the caller alone: its token, seed and
    expiry, and no kind). No `rules_changed`: the page reads none of the boxes' rules. No box taken: the public feed of
    catches went with the cut (the owner's call, 2026-11-15), since the page had shown nothing of it since 2026-11-06.
    A stream's session reader refuses nothing. **A box goes only to a caller with this stream open.**
- **Events** (`events.proto`): `BoxClaimed`, with `bonuses.v1.BonusClaimed`'s two fields and numbers, so a payload the
  bonuses module told reads as the new type, plus `token` (it names the catch) and `welcome` (a newcomer's early box);
  and the box facts `BoxCaught`, `BoxLapsed`, `ForeignBoxClaimed`, with `bonuses.v1`'s fields and numbers.
- **`InternalService`**: `PlayRuleSet`, `GetRuleSetNames`, `GetRuleSet`, for the gamemaster and promo.
- **`AdminService`**: `GetRules` and `SaveRuleSet`, which the backoffice's Box & quiz rules page and Claude call
  (`boxes_get_rules`, `boxes_save_rule_set`; [Claude as a connector](connector.md)).

## What it asks of planet

| port | planet's answer | when |
|---|---|---|
| activity (`Registry.Clicked`) | `planet.v1.ClickTried`, group `bonuses-clicks` | every click planet tried |
| the map's freeze | `planet.v1.InternalService/GetFreeze` | each claim |

- **Activity.** The clicks taken, struck or unchanged ([What `Clicked` hears](offers-and-wallet.md#what-clicked-hears)).
  **The group kept the bonuses module's name**, so it goes on where the bonuses module left the stream, and no group is
  left behind. A click with no account (`auth.enabled` off) is not heard, as the quizzes do not hear it: nothing could
  pay a box to it.
- **No room.** **The boxes ask no hand what it holds**, and offer a box to every caller due, whatever the hands behind it
  hold. Whether a box pays anything is the bonuses module's to say when it is caught, by what planet's wallet keeps of
  the kind it draws (since 2026-11-16 the draw ignores the hand; [A box caught](bonuses.md#a-box-caught)), as it says for a right answer since the quizzes left
  ([The quizzes](quizzes-module.md)). Keeping "no box for a full hand" would need the bonuses' weights, and the boxes
  cannot ask the bonuses module, which hears them (a cycle, which `cmd/api/modules_test.go` refuses); asking planet's
  `GetRoom` instead would teach the boxes planet's kinds and still miss the kinds weighed 0. The owner's call,
  2026-11-15: a full hand is offered boxes that pay nothing.
- **The freeze**: as the bonuses module asked it ([The freeze](#the-freeze)).

## Its rule sets

**The boxes keep rule sets of their own**, named like planet's and played by name by the gamemaster beside planet's, the
antibot's, the bonuses' and the quizzes' ([The gamemaster](gamemaster.md)). `default` holds every value, another set only
what it changes. `boxes.v1.Rules`, each value named by its path (`minInterval is 0s, and must be above 0`):

- `minInterval`, `maxInterval` — a box's wait
- `maxBoxesPerHour` — no box for a caller that caught this many in the last hour (the bonuses module's
  `boxes.maxChargesPerHour`, the same number: [The hourly cap](#the-hourly-cap))
- `offerTtl` — how long a box flies; it must outlast the flight the page draws
- `missRetry` — a box missed brings the next one this soon, once
- `activeWithin` — no box for a caller that has not clicked for this long
- `welcome.within`, `welcome.boxAfter` — a newcomer's early box ([A newcomer's first box and first quiz](welcome-bonus.md))
- `frozen` — no box is offered, and every claim is refused

**Promo cites them as `boxes.<path>`** ([Promo](promo.md)): promo's migration `20261115104725_boxes_paths` renamed the
stored cards' `{bonuses.boxes.…}`, `{bonuses.offerTtl}`, `{bonuses.missRetry}`, `{bonuses.activeWithin}` and
`{bonuses.welcome.…}`; production's urf card named two of them. Each set taken is logged at Warn (`box rules in force`),
each save too (`admin box rule set saved` or `refused`).

**Where the sets came from.** Migration `20261115103905_from_bonuses` copies each of the bonuses module's sets, under the
same name: its `boxes` block (`maxChargesPerHour` becomes `maxBoxesPerHour`), and its `offerTtl`, `missRetry`,
`activeWithin`, `welcome` and `frozen`. A set with none of them comes across empty, so it plays default's. On a database
with no bonuses schema it seeds production's box half of 2026-11-15. The bonuses module's sets keep those keys for this
release, read no more (`bonuses.v1.Rules` reserves them), so the copy reads them whichever module migrates first; a later
release strips them.

## The freeze

**A claim is refused before the box is spent** (`boxesv1controller.NewFreezeInterceptor`), with `FailedPrecondition` and a
`FrozenRefusal`, while the boxes play a frozen set or planet's `GetFreeze` answers frozen: its rules in force freeze the
map, or an arena's freeze holds it ([Arena battles](arenas.md)). A slot that comes due while frozen is lost: the next wait
is drawn.

## A box caught, told

`claim_box_usecase` spends the box in the registry and **records `BoxClaimed` in its outbox**, in a transaction of its own,
for a caller with an account. A caller with no account is told nothing: nothing pays it.

- **The outbox**, because a box caught told is a box caught paid: a `BoxClaimed` Redis refused would be a charge lost,
  where the bonuses module paid it at once before. The relay sends every second, so the reward reaches the page about a
  second after the catch ([Telling other modules what happened](composite-layer.md#what-may-not-be-lost-the-outbox)).
- **A record that fails answers the call with an error**, as a failed payment did: the box is spent.
- **Heard by** the bonuses module (`bonuses-boxes`, which pays it, new stream only), [missions](missions.md)
  (`missions-boxes`) and [progression](progression.md) (`player-stats-bonuses`). The last two kept their groups and read
  what the bonuses module's stream still held first (`cpredis.SubscribeMoved` from `bonuses.v1.BonusClaimed`).
- **It is told at the catch, not at the payment.** `BonusClaimed` was told once planet had kept the charge; a box that
  pays nothing now (below) still counts as a box caught for missions and the `Loot` titles.

## The box facts

`BoxCaught`, `BoxLapsed` and `ForeignBoxClaimed`, for the [antibot](antibot.md), through the module's own facts log and
its 100 ms relay: each at a later time than the one before, so the antibot's marks know one told again. **The log's first
time is past the bonuses module's last**: both read the wall clock, and the new process starts after the old one stopped.
The antibot's groups (`antibot-boxes-caught`, `-lapsed`, `-foreign`) kept their names and read what the bonuses module's
streams held first (`cpredis.SubscribeMoved`).

## What the cut changed for a player

**What a box pays did not change, but for two cases** ([A box caught](bonuses.md#a-box-caught)):

1. **A box may pay nothing.** The bonuses module offered a box only for a kind the hand had room for and the rules
   weighed. The boxes know no kind and ask no room ([No room](#what-it-asks-of-planet)), so a box comes whatever the hand
   holds: a full hand, or one with room only for a kind weighed 0 (capital-siege weighs spread and enclose 0), is offered
   a box that pays nothing; the page then names the kind drawn (it said the inventory is full until 2026-11-17), and `bonus_box_rewards_total{outcome="nothing"}`
   counts it. So is a newcomer's early box when the set weighs spread 0 or the spread pool is full: before, it gave no
   early box at all. A full hand's catches now count, as any catch, for the box missions, the `Loot` titles and the
   antibot's box facts, where before it was offered nothing to catch.
2. **The hourly cap counts boxes caught**, not charges granted. Each box paid one grant, so the numbers are the same.

The claim's answer no longer says what the box held: the page shows the reward from planet's `charges_granted`, about a
second after the catch.

## Metrics

Every metric kept its name: `bonus_claims_total{outcome}` (`granted` now means the box was taken), `bonus_offers_total`,
`bonus_lapsed_total`, `bonus_catch_seconds`, `bonus_claims_foreign_total` and `bonus_session_checks`.

## The deploy

A hard switch ([A server restart reloads the page](../../../docs/reload-on-restart.md)): the page reads `BoxService` and
the box's reward from `charges_granted`. **The page first, then the server**, as always. After it:

- `XGROUP DESTROY` of `missions-boxes` and `player-stats-bonuses` on `events:bonuses.v1.BonusClaimed`, and of
  `antibot-boxes-caught`, `antibot-boxes-lapsed` and `antibot-boxes-foreign` on `events:bonuses.v1.BoxCaught`,
  `BoxLapsed` and `ForeignBoxClaimed`, once each read what it left there; then a `DEL` of the four streams.
- The bonuses module's box keys in `bonuses.rule_sets` stay until a later release strips them.

## Tests

The registry's (the schedule, the welcome, the pace, the frozen set, a click heard late, a click with no account, the
entrants), the use cases', the handlers', the stream's and its sink's, the facts log's, the rules package's (and
`cprules`' contract suites on its migrations, and the copy of production's sets from the bonuses module), and promo's
migration test. **One e2e smoke test walks a box's whole way** (`e2e/box_test.go`): a guest's stream is offered a box, the
guest catches it, and planet tells the grant on the guest's own stream, so the boxes' outbox, the bonuses' subscription,
the room, the draw and planet's grant are wired. It plays the boxes' `default` at one box a second, and puts it back.
