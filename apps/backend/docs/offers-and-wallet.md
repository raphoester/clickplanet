# Offers and the wallet: the seam, then the cut

**Box and quiz offers left planet for a `bonuses` module of their own** ([The offers](bonuses.md)), downstream of planet
like `missions`, while planet knows nothing of it. The quizzes are to carry a difficulty, with players rated on the
same scale, and that work must not grow `planet`, the biggest module. **The rule of the cut: granting is the offers', holding and spending
is planet's.** The wallet, the charges a player holds and spends, stays in planet.

- **Step A (done, 2026-11-11)** cut the seam *inside* planet, with no change to the wire, the stored data or the game,
  so that B is mostly a move. The sections below are that seam, as B kept it.
- **Step B (done, 2026-11-11)** moved the offers out: `internal/bonuses/`, `proto/bonuses/v1` (`ClaimBonus`,
  `OpenQuiz`, `AnswerQuiz`, a stream of its own, `BonusClaimed`, `QuizAnswered` and the box facts), rule sets of its own
  that the gamemaster plays by name like the antibot's. Each port became a loopback call to planet, in planet's words
  (below). `ClickTried` gained `dry`, planet's `BonusRules` shrank to `ChargeRules` (`Rules.charges`), and
  `GetBonusRules` became `GetChargeRules`. A hard-switch deploy ([A server restart reloads the
  page](../../../docs/reload-on-restart.md)). See [What B did beside the plan](#what-b-did-beside-the-plan).
- **Step C** was the quiz difficulty, inside `bonuses` (2026-11-11).
- **Then the quizzes left `bonuses`** (2026-11-13) for a module of their own, the same way: a seam, then a move
  ([The quizzes](quizzes-module.md)). A right answer stopped paying at once: the offers pay it from `QuizAnswered`, and
  `GetChargesOf`, which answered the hand a wrong answer leaves, went. The sections below name the quiz as B left it.

## The two sides

Since B the offers are `internal/bonuses/internal/bonuses` and the bank `internal/bonuses/internal/quizzes`, moved as they
were; the paths below are A's.

- **The wallet, `planet/internal/charges/`**: `Kind` as the wallet knows it, `Holder`, `Held` (`Granted`, `After*`,
  `Count`, `Full`), `Charges` and its `Store`, `GrantID`, `Bombing`, `Terrain` and `Pocket`, the stores
  (`postgres_charge_store`, `postgres_grant_store`, `inmemory_grant_store`), the wallet's half of the rules, and the
  use cases that spend or grant from planet's own endpoints: `drop_bomb`, `use_refill`, `place_shield`, `get_charges`,
  `grant_charges` (the operator's), `apply_grant` (the `planet.v1.GrantCharges` command) and `pay` (what an offer
  granted). See [Charges](charges.md).
- **The offers, `planet/internal/bonuses/`**: the `Registry` (`bonus.go`, `quiz.go`, `welcome.go`, `entrant.go`,
  `config.go`), `box_facts`, the offers' half of the rules, and the use cases `claim_bonus`, `open_quiz`,
  `answer_quiz` and `grant_sign_in_refill`. **`planet/internal/quizzes/`**, the bank and its draw, sits beside it and
  only the offers use it.
- **`planet/internal/notices/`**: the per-caller channel planet keeps for what is its own (below).

**`bonuses` and `quizzes` imported no package of planet's but each other**, tests included
(`bonuses/seam_test.go`: `TestTheOffersImportNoConceptOfPlanets`, and `TestTheQuizzesAreTheOffersAlone`). Since B, Go's
own `internal/` rule keeps them from planet's packages, `cmd/api/modules_test.go` pins what the module speaks, and
`TestTheQuizzesAreTheOffersAlone` lives on inside the module. Everything they need of planet
comes through a port they declare, answered in `internal/planet/wallet.go` (the wallet's ports), `module.go` and the
controller (`planetv1controller/offercaller`). **`bonuses` keeps its own `Kind`** (what a box can be drawn as, with its
weights and its amounts), mapped to the wallet's by name: the duplication is on purpose, since the offers speak
planet's proto kinds (`planetwords`). `TestTheOffersAndTheWalletNameTheSameKinds` pins the names. In A `bonuses.Hand`
was the wallet's `charges.Held` field for field, converted in place; since B the hand crosses as `planet.v1.ChargesHeld`.

## The ports `bonuses` declares

| port | A answered it from | B answers it with |
|---|---|---|
| the caller | `offercaller.Of(ctx)` | the module's own session interceptors (`bonusesv1controller/caller`) |
| `Registry.Clicked` | `bonus_click`, in planet's click chain | `planet.v1.ClickTried`, group `bonuses-clicks` |
| `Room.Full` | `wallet.go` | `planet.v1.InternalService/GetRoom` |
| `Wallet.Pay` | `pay_usecase` | `planet.v1.InternalService/GrantCharges` |
| `Wallet.Hand` | `charges.Charges.Held` | `planet.v1.InternalService/GetChargesOf` |
| `box_facts.Facts` | planet's `facts.Log` | the module's own `facts.Log` |
| `quizzes.Shares` | planet's tile map | `planet.v1.InternalService/GetTerritories`, every 10 s |

- **The caller (`bonuses.Caller`)**: scope, account, linked. The offers read no request: the handler hands it in
  (`offercaller.Of(ctx)`, from `clicks.PayerOf`), in each use case's `In`. `EntrantOf(caller)` is the schedule's key
  as before. Planet keeps its own word for the same thing, `clicks.CallerOf(payer)`, for a stream's `yours` and the
  notices.
- **Activity in: `Registry.Clicked(Click)`**, fed by `clicks/usecases/click_usecase/bonus_click`. A `Click` carries
  only what `ClickTried` says of its caller (scope, account, linked), plus `Dry`, the bank left empty, and `Born`, the
  account's age read off its id (`Payer.Created`).
- **Room: `Room.Full(ctx, accounts)`**, the kinds each account has no room for (below).
- **Pay: each use case's `Wallet.Pay(ctx, Payment)`**, and `Wallet.Hand(ctx, account)` for the hand a wrong answer
  leaves (below).
- **The box facts: `box_facts.Facts.Tell(Fact)`**, `Fact` being the func type itself; planet's `facts.Fact` is an alias
  of it, so its log meets the port as it is.
- **The lean: `quizzes.Shares`**, answered by planet's tile map; in B by planet's `GetTerritories`.

### What `Clicked` hears

**Exactly the clicks `ClickTried` tells as `TAKEN`, `STRUCK` or `UNCHANGED`.** `bonus_click` sits outside the throttle
and inside the reach, the allegiance and the refusal (see [Inside the planet module](planet-module.md)), and calls
`Clicked` only when the chain inside it answered no error. So a try that is `REFUSED`, `OTHER_FLAG` or `OUT_OF_REACH`
never reaches it, and one that is `THROTTLED`, `INVALID` (an unknown flag, a tile off the map, two bonuses together)
or `FAILED` (a spread or an enclose that failed after the take) reaches it with an error and is not heard.
`TestOnlyATakeAStrikeOrAnUnchangedTileIsHeard` pins it. **Since B, `click_tried_subscriber` keeps exactly those three
outcomes** and reads `dry` off the try (planet's `try_click` tells it), and its own
`TestOnlyATakeAStrikeOrAnUnchangedTileIsHeard` pins it. What the relay's delay does to it is in [The
offers](bonuses.md#what-it-asks-of-planet-in-planets-words).

### Room

**The registry asks the wallet outside its lock**, since it is a loopback call (B: `GetRoom`). A sweep is three steps:
`dueCallers` tidies the registry and finds who is due a quiz or a box, with the accounts playing behind each, under the
lock; `ask` calls `Room.Full` once per caller due, unlocked; `offerDue` takes the lock again and offers the quizzes,
then the boxes, each to a caller **still due** (`quizDue`, `due`) and still played by the accounts asked of
(`playedBy`). An account whose first click lands while the room is asked waits for the next sweep, a second later, so a
newcomer's welcome is not lost to the race (`TestAnAccountThatFirstClicksWhileTheRoomIsAskedWaitsForTheNextSweep`).
The outcomes are those of before: the same kinds left out, and an error still loses the slot, as an away player's
(`TestARoomTheWalletCannotTellLosesTheSlot`). The wallet applies its own sizes (`Held.Full(charges.Rules)`); the offers
never see one. Since B, `Held.Full` also counts a bomb while `charges.bomb.off`, so the offers need no `bomb.off`.
`TestTheRoomIsAskedWithTheRegistryUnlocked` pins the lock.

### Pay

**`Pay` grants once per `GrantID`, which `bonuses` supplies**, and answers what was kept and the hand after, which
`ClaimBonusResponse` and `AnswerQuizResponse` carry as before. A payment told again keeps nothing and answers the hand
as it is.

- **The ids**: a box's or a quiz's is its token (`OfferGrantOf`, a UUIDv5 of it); the sign-in refill's is the
  account's (`SignInGrantOf`, the same id for an account as before the cut, pinned by a test).
- **The path is the `GrantCharges` command's**: `charges/usecases/pay_usecase` records the id and grants in one
  transaction, as `apply_grant_usecase` does, and answers. Since B it is `planet.v1.InternalService/GrantCharges`.
- **Where an id is kept**: the refill's in `charge_grants`, as before. An offer's **in memory**, for 10 minutes
  (`inmemory_grant_store`): the offer itself is memory only, and nothing of a box or a quiz is written to postgres
  that was not before. Kept in `charge_grants`, an offer's id would be a row per box caught and per right answer, about
  90 bytes with its key, never removed: the table has no index on `granted_at` and nothing to tell a catch's row from a
  refill's, which must stay, so bounding it needs a migration. B kept it in memory (the owner's call): both modules
  share one process, so a retry never outlives a restart. The call says which with `lasting`.
- **A failed payment loses the box or the answer**, as a failed grant did. The id an offer recorded in memory stays,
  so in B a retry after a failure that came back is answered as told again; a retry after a lost answer is right.

## The per-caller channel

Since B the offers have a stream of their own (`bonuses.v1.BonusService/ListenForEvents`), and planet's carries only
its own; the rest of this section is A's.


The registry used to carry two things that are planet's: a spread (`PublishSpread`, from `spread_click`) and a grant
(`PublishGranted`, from `apply_grant` and `grant_charges`, `charges_granted` on the stream). **They have a channel of
their own, `notices.Board`**: a spread to every stream open, a grant only to the streams of the signed-in account it
was made to, as before. The registry carries the offers alone: `bonus_offered`, `quiz_offered`, `bonus_taken`, still
only to callers with a stream open. `listen_for_events_usecase` attends both and merges them into the one
`ListenForEvents` stream. The wire did not change; each channel keeps its own order, and nothing ordered a spread or a
grant against an offer before. Each buffers 32 events per stream and drops what a stream that is not reading cannot
take.

## The rules split

Since B each half is its module's: the offers' is `bonuses.v1.Rules` in `bonuses.rule_sets`, the wallet's is
`planet.v1.Rules.charges` (`ChargeRules`), and `bomb.off` is the wallet's alone. In A:

**Both halves read the same `planet.v1.BonusRules` and the same stored sets**, each its own part
([Rules](rules.md)), each validated with the messages of before:

- **The offers' (`bonuses.Rules`, `parts.Grants`)**: `boxes`, `kinds`, `offerTtl`, `missRetry`, `activeWithin`,
  `spread.maxPerBox`, `enclose.maxPerBox`, `shield.minPerBox`, `shield.maxPerBox`, `welcome`, `quiz`, `signInRefill`,
  and `bomb.off`.
- **The wallet's (`charges.Rules`, `charges.Part`)**: `spread.clicks`, `bomb.rings`, `enclose.maxTiles`,
  `enclose.held`, `enclose.unlimited`, `shield.held`, `shield.perTile`, and `bomb.off`.

**`bomb.off` is read by both**: the offers offer no bomb while it is on (and refuse a set whose only kind is a bomb), the
wallet drops none and `GetBonusRules.bombs_off` tells the client. Capping a grant at what a hand holds is the wallet's.
`GetBonusRules` reads the wallet's half, and of the offers' half `signInRefill` alone, passed to it on its own; its
rename is B's.

## Events and metrics

In A the offers still published `planet.v1.BonusClaimed`, `QuizAnswered` and the box facts. Since B they are
`bonuses.v1`'s, with the same fields and numbers, and each listener's group drains what planet's stream still held
before it reads the new one (`cpredis.SubscribeMoved`). Every metric keeps its name (`bonus_claims_total`,
`bonus_offers_total`, `quiz_answers_total` and the rest).

## What B did beside the plan

- **`planet.v1.InternalService/GetFreeze`**, a fourth call: planet's freeze interceptor refused `ClaimBonus`,
  `OpenQuiz` and `AnswerQuiz` while the frozen set played or an arena's freeze held the map. The offers have a frozen
  set of their own for the first, but an arena's freeze is planet's alone, so they ask planet at each claim, opening
  and answer. Without it a box caught in an arena's last seconds would have been granted.
- **`GrantCharges` takes `lasting`**: the offers' ids stay in memory as planned, but the sign-in refill's must stay in
  `charge_grants`, where every refill before the move is, or a `PlayerCreated` told again would pay twice.
- **The offers' stream also carries `rules_changed`**, so an open page reads `GetGrantRules` again when the offers take
  another set, as planet's `rules_changed` sends it to `GetChargeRules`.
- **The offers' `frozen` offers nothing**: in A a frozen planet still offered boxes and refused their claims.
- **A click is timed by its try**, not by when it is heard, and one heard after `activeWithin` is dropped: a backlog the
  group reads at a boot is no sign of play.
- **Planet keeps `bonuses` in its stored sets** for this release, so the offers' first migration copies it whichever
  module migrates first; planet reads it no more (`Rules.bonuses` is reserved), and a later release strips it.
- **`cpredis.SubscribeMoved`**, so no box, answer or box fact told before the deploy is lost or counted twice.
- **Production's pools**: the module takes one connection, and promo goes from two to one, to stay 10 under
  `max_connections` without restarting postgres at the deploy.
- **`bonus_session_checks`** counts the three calls' session verdicts, which `click_session_checks` counted before.
