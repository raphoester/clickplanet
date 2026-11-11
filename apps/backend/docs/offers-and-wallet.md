# Offers and the wallet: the seam inside planet

**Box and quiz offers are leaving planet for a `bonuses` module of their own**, downstream of planet like `missions`,
while planet knows nothing of it. The quizzes are to carry a difficulty, with players rated on the same scale, and that
work must not grow `planet`, the biggest module. **The rule of the cut: granting is the offers', holding and spending
is planet's.** The wallet, the charges a player holds and spends, stays in planet.

- **Step A (done, 2026-11-11)** cut the seam *inside* planet, with no change to the wire, the stored data or the game,
  so that B is mostly a move. This doc is that seam.
- **Step B** moves the offers out: `internal/bonuses/`, `proto/bonuses/v1` (`ClaimBonus`, `OpenQuiz`, `AnswerQuiz`, a
  stream of its own, `BonusClaimed`, `QuizAnswered` and the box facts), rule sets of its own that the gamemaster plays
  by name like the antibot's. Planet gains an internal synchronous, idempotent `GrantCharges` and a "can this account
  take more of a kind" call, and `ClickTried` gains `dry`. Planet's `BonusRules` shrinks to `ChargeRules`, and
  `GetBonusRules` becomes `GetChargeRules`. A hard-switch deploy ([A server restart reloads the
  page](../../../docs/reload-on-restart.md)).
- **Step C** is the quiz difficulty, inside `bonuses`.

## The two sides

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

**`bonuses` and `quizzes` import no package of planet's but each other**, tests included
(`bonuses/seam_test.go`: `TestTheOffersImportNoConceptOfPlanets`, and `TestTheQuizzesAreTheOffersAlone`, which lets
no package of planet's but the offers, and the module that loads it, import the bank). Everything they need of planet
comes through a port they declare, answered in `internal/planet/wallet.go` (the wallet's ports), `module.go` and the
controller (`planetv1controller/offercaller`). **`bonuses` keeps its own `Kind`** (what a box can be drawn as, with its
weights and its amounts), mapped to the wallet's by name: the duplication is on purpose, since in B the offers speak
planet's proto kinds. `TestTheOffersAndTheWalletNameTheSameKinds` pins the names. `bonuses.Hand` is the wallet's
`charges.Held` field for field, so the conversion between them compiles only while they agree.

## The ports `bonuses` declares

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
`TestOnlyATakeAStrikeOrAnUnchangedTileIsHeard` pins it. **In B, `bonuses` must filter `ClickTried` the same way**,
and read `dry` off it.

### Room

**The registry asks the wallet outside its lock**, since in B it is a loopback call. A sweep is three steps:
`dueCallers` tidies the registry and finds who is due a quiz or a box, with the accounts playing behind each, under the
lock; `ask` calls `Room.Full` once per caller due, unlocked; `offerDue` takes the lock again and offers the quizzes,
then the boxes, each to a caller **still due** (`quizDue`, `due`) and still played by the accounts asked of
(`playedBy`). An account whose first click lands while the room is asked waits for the next sweep, a second later, so a
newcomer's welcome is not lost to the race (`TestAnAccountThatFirstClicksWhileTheRoomIsAskedWaitsForTheNextSweep`).
The outcomes are those of before: the same kinds left out, and an error still loses the slot, as an away player's
(`TestARoomTheWalletCannotTellLosesTheSlot`). The wallet applies its own sizes (`Held.Full(charges.Rules)`); the offers
never see one. `TestTheRoomIsAskedWithTheRegistryUnlocked` pins the lock.

### Pay

**`Pay` grants once per `GrantID`, which `bonuses` supplies**, and answers what was kept and the hand after, which
`ClaimBonusResponse` and `AnswerQuizResponse` carry as before. A payment told again keeps nothing and answers the hand
as it is.

- **The ids**: a box's or a quiz's is its token (`OfferGrantOf`, a UUIDv5 of it); the sign-in refill's is the
  account's (`SignInGrantOf`, the same id for an account as before the cut, pinned by a test).
- **The path is the `GrantCharges` command's**: `charges/usecases/pay_usecase` records the id and grants in one
  transaction, as `apply_grant_usecase` does, and answers.
- **Where an id is kept**: the refill's in `charge_grants`, as before. An offer's **in memory**, for 10 minutes
  (`inmemory_grant_store`): the offer itself is memory only, and nothing of a box or a quiz is written to postgres
  that was not before. Kept in `charge_grants`, an offer's id would be a row per box caught and per right answer, about
  90 bytes with its key, never removed: the table has no index on `granted_at` and nothing to tell a catch's row from a
  refill's, which must stay, so bounding it needs a migration. That is the owner's call for B.
- **A failed payment loses the box or the answer**, as a failed grant did. The id an offer recorded in memory stays,
  so in B a retry after a failure that came back is answered as told again; a retry after a lost answer is right.

## The per-caller channel

The registry used to carry two things that are planet's: a spread (`PublishSpread`, from `spread_click`) and a grant
(`PublishGranted`, from `apply_grant` and `grant_charges`, `charges_granted` on the stream). **They have a channel of
their own, `notices.Board`**: a spread to every stream open, a grant only to the streams of the signed-in account it
was made to, as before. The registry carries the offers alone: `bonus_offered`, `quiz_offered`, `bonus_taken`, still
only to callers with a stream open. `listen_for_events_usecase` attends both and merges them into the one
`ListenForEvents` stream. The wire did not change; each channel keeps its own order, and nothing ordered a spread or a
grant against an offer before. Each buffers 32 events per stream and drops what a stream that is not reading cannot
take.

## The rules split

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

Unchanged: the offers still publish `planet.v1.BonusClaimed`, `QuizAnswered` and the box facts (`BoxCaught`,
`BoxLapsed`, `ForeignBoxClaimed`) as before, and every metric keeps its name (`bonus_claims_total`,
`bonus_offers_total`, `quiz_answers_total` and the rest).
