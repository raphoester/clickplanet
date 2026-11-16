# Charges (refill, bomb, enclose, spread, shields, virus, mine, rally)

**The charges are the wallet, `internal/planet/internal/charges/`**: what an account holds and spends. What grants them,
a box caught, a right answer or a guest's first sign-in, is the bonuses module's, which reaches the wallet only through
planet's internal calls, in planet's words (`GrantCharges`, `GetRoom`): see [The bonuses](bonuses.md) and [Offers and the
wallet](offers-and-wallet.md). The boxes are the boxes module's ([The boxes](boxes-module.md)).

A timer rewarded speed rather than planning: with a bank of clicks, a 10s spread
let a player dump the whole bank at seven tiles a click, more than a bomb, and a
bomb held 30s was dropped on the first target in sight. So every kind is a
**charge**, kept until the player uses it: nothing lapses and nothing is used on
its own.

- **Only an account holds a charge.** `charges.Holder` is the account the click
  token names, and `HolderOf(clicks.PayerOf(ctx))` derives it, so the click chain, the drop and the shield agree on
  whose charge it is; the offers pay the account the same token names (`bonuses.Caller`, which the handler reads). A charge is the account's
  so it survives closing the tab, a new address and another device. A caller with
  no account is `NoHolder`: it holds nothing, and a scope where no account plays
  is offered nothing. Every client mints a guest account, so this leaves
  out only a caller with no token at all.
- **The rules are a value, `charges.Held`**: a refill and a bomb (held or not), a
  stack of enclosures, a pool of spread clicks and a stack of shields. `Granted`,
  `AfterRefill`, `AfterBomb`, `AfterEnclose`, `AfterSpreadClick` and `AfterShield`
  build a new `Held` and change
  nothing; `charges.Charges` writes it. `Count(kind)` is how many of a kind are held,
  and `Full(rules)` the kinds a hand has no room for, a bomb while bombs are off and an enclosure while enclosures
  are off included, which
  `InternalService/GetRoom` answers. The sizes are the wallet's rules (`charges.Rules`, `charges.*` in a set); the
  offers never read one.
- **A rule set may make enclosures unlimited** (`charges.enclose.unlimited`): enclose then spends no charge, and
  the stack held is left as it is. See [Unlimited enclosures](enclose.md#unlimited-enclosures-chargesencloseunlimited).
- **A rule set may turn enclosures off** (`charges.enclose.off`): an enclose click closes nothing and spends nothing,
  no hand has room for one, and the stack held is kept. See [Enclosures off](enclose.md#enclosures-off-chargesencloseoff).
- **A rule set may make spreads unlimited** (`charges.spread.unlimited`): spread then spends no charge, and the pool
  held is left as it is. See [Unlimited spreads](spread.md#unlimited-spreads-chargesspreadunlimited).
- **How much fits.** A refill, a bomb and a virus are one: a second replaces the first,
  which is what stops a stockpile of bombs being dropped all at once. Enclosures
  stack to the rule set's `charges.enclose.held` (3), spread clicks pool to `charges.spread.clicks`
  (8), shields stack to `charges.shield.held` (30) mines to `charges.mine.held` (3) and rallies to `charges.rally.held` (3) (see [Rules](rules.md)). A box draws its amount evenly from 1 to
  the bonuses' `maxPerBox`, or from their `shield.minPerBox` for shields (`bonuses.amountOf`, `crypto/rand`), and the
  grant caps it at the size: the cap is the wallet's. **What was kept** reaches the page as the grant's `gained`, on
  planet's stream (`charges_granted`, below): `Charges.Grant` answers the hand before and after it (`charges.Gain`), both
  read under the lock of its one transaction, so a player is never told of clicks that did not fit, and a spend in
  another tab cannot change the number. The bonuses module's `InternalService/GrantCharges` runs the wallet's payment
  (`pay_usecase`). Until 2026-11-15 the claim's own answer said what was kept.
- **The schedule offers a box whatever the hand holds** (since 2026-11-15). The schedule is by entrant and a charge
  is by account, and the boxes know no kind and ask no room ([The boxes](boxes-module.md#what-it-asks-of-planet)). The
  kind is drawn at the catch, among the kinds the claiming account's hand has room for then, and a full hand is paid
  nothing ([A box caught](bonuses.md#a-box-caught)).
- **Off by default, one at a time.** `ClickRequest.spread` and `enclose` say what
  the player switched on for this click, and `click_usecase.In` carries them.
  `spread_click` spends a spread click only with `Spread`, once the click is accepted (see [Spread](spread.md)),
  `enclose_click` an enclosure only with `Enclose`. **Both at once is refused** by the rule itself
  (`clicks.ErrBonusesTogether`, answered `InvalidArgument`) before anything is
  written or spent: a spread's tiles and an enclose's pocket would each take what
  the other decides. The client keeps one on at a time too, and aiming the bomb
  switches both off.
- **Another module grants with a command**, `planet.v1.GrantCharges` on `commands:planet.v1.GrantCharges`
  (`commands/apply_grant_handler`, `apply_grant_usecase`): the grant and its `grant_id` (`planet.charge_grants`) are
  kept in one transaction, so a command told twice grants once. It caps at the stack as any grant does, and answers
  nothing. `missions` paid its rewards this way until missions paid XP; nothing sends it now (see
  [Missions pay XP](missions.md#missions-pay-xp)).
- **Or with a call answered at once**, `InternalService/GrantCharges{grant_id, account_id, kind, amount, lasting}`
  (`internal_grant_charges_handler`, then `pay_usecase`, the command's path with an answer: what the stack kept and
  the hand after). The bonuses module pays a box the boxes told caught, a right answer the quizzes told and a new player's refill this
  way. With `lasting`
  the grant id is kept for good in `charge_grants`, as a command's: the refill, once per account, from player's
  `PlayerCreated` ([A guest's first sign-in](sign-in-refill.md)). Without it, ten minutes in memory
  (`inmemory_grant_store`): a box's or an answer's id is its token's, which lives minutes and never outlives a
  restart. A bad id or account, an unknown kind or no amount is `InvalidArgument`.
- <a id="a-grant-on-the-stream"></a>**A grant reaches the player at once, on the planet stream it has open**:
  `charges_granted`, all it holds after the grant and **what the grant added** (`gained`, each kind and how many the
  stack kept), sent once it is committed (`notices.Board.PublishGranted`, a `notices.Event`'s `Granted`, a
  `charges.Gain`), from `apply_grant_usecase`, the operator's `grant_charges_usecase` and, since 2026-11-13, the offers'
  `pay_usecase`, whose grant told again sends nothing. **It reaches every stream whose session names the account
  granted, a guest's too** (`notices.Board.Attend` keys a stream by the account its token names): a guest's token names
  its own account, which the others behind its address do not share. A stream with no token hears none. The page shows
  a right answer's reward and a box's from it, since `AnswerQuiz` and `ClaimBox` pay nothing ([The
  quizzes](quizzes-module.md), [The boxes](boxes-module.md)). Delivery is at
  most once; the client reads the charges again when its stream comes back.
- **Nothing lapses.** Migration `20260919120000_charges_never_lapse` replaced the
  `*_until` columns with `refill` and `bomb` booleans and an `enclosures` count,
  keeping what was still in time. `bonus.chargeTTL` is gone.
- **No memory copy: every grant, spend and read goes to postgres**, one row per
  account in `planet.charges` (`refill`, `bomb`, `enclosures`, `spread_clicks`,
  `shields`, `virus`, `mines`, `rallies`). `charges.Charges` holds the sizes and the `Store` port, whose `Change`
  runs one transaction under `pg_advisory_xact_lock` on the account. An empty hand
  is no row. A failing database fails the call; a spread click fails with its tile
  taken. A deleted account's row stays until something removes it. The boxes'
  state (schedules, offers, the hourly caps) is memory only, in the boxes module, so
  a restart gives everyone a fresh schedule.
- **Each spend is atomic**: `Charges.Spend` checks and takes in one `Change`, so two
  tabs racing for the last one get one, and the other `charges.ErrNotHeld`.
  `charges.StoreContractSuite` runs on postgres and on `MemoryStore` (behind the tag).
- **A box is taken from memory before it is told caught**, so a record that fails (the boxes' database down) loses it.
  That failure is not a refusal: `NotFound` is only for a token that is not the caller's or not there, and a failed
  record goes to the error net (logged, answered `internal`). `bonus_claims_total` counts it `failed`, apart from
  `refused`. Once the box is taken, the record does not stop when the page leaves: it runs on `context.WithoutCancel` for
  at most 5 s. A box caught is paid from the boxes' `BoxClaimed`, and a right answer from the quizzes' `QuizAnswered`,
  which the bus tries again until planet answers.
- **A return undoes a spend**: `Charges.Return` puts back what a spend took, and reads no rule, since it grants
  nothing. Only a shield the map refused comes back this way (see [A shield's way](placements.md)).
- **Only a grant pushes them** (`charges_granted`, above). Otherwise they are not live news: `GetCharges` answers what the
  caller holds, read by the client at load and when its account changes, and
  `PlaceShieldResponse.charges` answers a
  shield placed. After that the client follows
  its own calls: a drop spends the bomb, an accepted click sent with spread on a
  spread click (not while spreads are unlimited), its own `tiles_enclosed` (`yours`) an enclosure. A charge spent in another
  tab shows until the next read. **`Click` answers nothing about them.** It was so
  a shadow-banned caller could not tell its clicks were dropped; a ban is told now
  (see [A ban is told](antibot.md#a-ban-is-told)), so the answer could carry them.
  `planetv1controller/chargesheld` encodes the message planet answers, `GrantCharges` and `GetChargesOf` included,
  which the offers answer on in planet's words (`planet.v1.ChargesHeld`, through `planetwords`).
- **How big a charge is, is a rule, not state.** `GetChargeRules` (`GetBonusRules` until 2026-11-11) answers the
  blast radius, the enclose's `maxTiles`, the spread pool's size and whether it is unlimited
  (`spread_unlimited`), and the
  enclosure stack's size, how many shields a player holds and a tile takes
  (`shields`, `tile_shields`), and every step of the toll (`toll_steps`, from
  `clicks.Toll.Steps`), all read from the rule set in force (see [Rules](rules.md)), so the client can show the
  whole table and the slowdown of a country it does not play for. It is
  `NO_SIDE_EFFECTS`, a GET no cache keeps: the client reads it at load and again on each `rules_changed`.
- **A stack held above a smaller size is kept** (`Held.Granted` never takes from a stack): a hand of 12 shields
  under a new stack of 5 keeps 12, is offered no shield box, and a grant adds none until it is under 5.
