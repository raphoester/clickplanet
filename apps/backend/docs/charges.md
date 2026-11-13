# Charges (refill, bomb, enclose, spread, shields)

**The charges are the wallet, `internal/planet/internal/charges/`**: what an account holds and spends. What grants them,
a box, a right answer or a guest's first sign-in, is the offers', the bonuses module since 2026-11-11, which reaches the
wallet only through planet's internal calls, in planet's words (`GrantCharges`, `GetRoom`, `GetChargesOf`): see
[The offers](bonuses.md) and [Offers and the wallet](offers-and-wallet.md).

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
  and `Full(rules)` the kinds a hand has no room for, a bomb while bombs are off included, which
  `InternalService/GetRoom` answers. The sizes are the wallet's rules (`charges.Rules`, `charges.*` in a set); the
  offers never read one.
- **A rule set may make enclosures unlimited** (`charges.enclose.unlimited`): enclose then spends no charge, and
  the stack held is left as it is. See [Unlimited enclosures](enclose.md#unlimited-enclosures-chargesencloseunlimited).
- **A rule set may make spreads unlimited** (`charges.spread.unlimited`): spread then spends no charge, and the pool
  held is left as it is. See [Unlimited spreads](spread.md#unlimited-spreads-chargesspreadunlimited).
- **How much fits.** A refill and a bomb are one: a second replaces the first,
  which is what stops a stockpile of bombs being dropped all at once. Enclosures
  stack to the rule set's `charges.enclose.held` (3), spread clicks pool to `charges.spread.clicks`
  (8) and shields stack to `charges.shield.held` (30) (see [Rules](rules.md)). A box draws its amount evenly from 1 to
  the offers' `maxPerBox`, or from their `shield.minPerBox` for shields (`bonuses.amountOf`, `crypto/rand`), and the
  grant caps it at the size: the cap is the wallet's. The claim answers
  `bonuses.v1.ClaimBonusResponse.amount` as **what was kept**: `Charges.Grant` answers the hand before and after it
  (`charges.Gain`), both read under the lock of its one transaction, so a player is never told of clicks that did not
  fit, and a spend in another tab cannot change the number. The offers get both from `InternalService/GrantCharges`
  (`kept`, `charges`), which runs the wallet's payment (`pay_usecase`).
- **The schedule offers no kind that is full** for any account that clicked as
  the entrant within `activeWithin` (`Registry.offerable`): the linked account
  itself, or every guest behind the scope. The schedule is by entrant and a charge
  is by account, so the offers hear both off every click planet tells (`planet.v1.ClickTried`: the scope, the
  account, linked), and ask planet's `GetRoom`, outside their lock, which kinds each of those accounts has no room
  for: a hand planet cannot tell is offered nothing (see
  [The offers](bonuses.md#what-it-asks-of-planet-in-planets-words)).
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
  nothing. `missions` pays its rewards this way.
- **Or with a call answered at once**, `InternalService/GrantCharges{grant_id, account_id, kind, amount, lasting}`
  (`internal_grant_charges_handler`, then `pay_usecase`, the command's path with an answer: what the stack kept and
  the hand after). The offers pay a box, a right answer and a new player's refill this way (2026-11-11). With `lasting`
  the grant id is kept for good in `charge_grants`, as a command's: the refill, once per account, from player's
  `PlayerCreated` ([A guest's first sign-in](sign-in-refill.md)). Without it, ten minutes in memory
  (`inmemory_grant_store`): a box's or an answer's id is its token's, which lives minutes and never outlives a
  restart. A bad id or account, an unknown kind or no amount is `InvalidArgument`.
- **A grant reaches the player at once, on the planet stream it has open**: `charges_granted`, all it holds after the
  grant, read in the grant's transaction and sent once it is committed (`notices.Board.PublishGranted`, a
  `notices.Event`'s `Granted`), from `apply_grant_usecase` and the operator's `grant_charges_usecase`. Only a signed-in
  account's own streams hear it: a guest's stream is its address's, which others share. Delivery is at most once; the
  client reads the charges again when its stream comes back. The sign-in refill sends none: the client reads the
  charges when it signs in.
- **Nothing lapses.** Migration `20260919120000_charges_never_lapse` replaced the
  `*_until` columns with `refill` and `bomb` booleans and an `enclosures` count,
  keeping what was still in time. `bonus.chargeTTL` is gone.
- **No memory copy: every grant, spend and read goes to postgres**, one row per
  account in `planet.charges` (`refill`, `bomb`, `enclosures`, `spread_clicks`,
  `shields`). `charges.Charges` holds the sizes and the `Store` port, whose `Change`
  runs one transaction under `pg_advisory_xact_lock` on the account. An empty hand
  is no row. A failing database fails the call; a spread click fails with its tile
  taken. A deleted account's row stays until something removes it. The offers'
  state (schedules, offers, the hourly caps) is memory only, in the bonuses module, so
  a restart gives everyone a fresh schedule.
- **Each spend is atomic**: `Charges.Spend` checks and takes in one `Change`, so two
  tabs racing for the last one get one, and the other `charges.ErrNotHeld`.
  `charges.StoreContractSuite` runs on postgres and on `MemoryStore` (behind the tag).
- **A box or a right answer is taken from memory before its grant**, so a grant that fails (the database down, or
  planet's `GrantCharges` failing) loses it. That failure is not a refusal: `NotFound` is only for a token that is not the caller's or not there, and a failed
  grant goes to the error net (logged, answered `internal`). `bonus_claims_total` and `quiz_answers_total` count it
  `failed`, apart from `refused`. Once the offer is taken, the grant does not stop when the page leaves: it runs on
  `context.WithoutCancel` for at most 5 s.
- **A return undoes a spend**: `Charges.Return` puts back what a spend took, and reads no rule, since it grants
  nothing. Only a shield the map refused comes back this way (see [A shield's way](placements.md)).
- **Only a grant pushes them** (`charges_granted`, above). Otherwise they are not live news: `GetCharges` answers what the
  caller holds, read by the client at load and when its account changes, and
  `bonuses.v1.ClaimBonusResponse.charges` answers the claim, `PlaceShieldResponse.charges` a
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
