# Charges (refill, bomb, enclose, spread, shields)

A timer rewarded speed rather than planning: with a bank of clicks, a 10s spread
let a player dump the whole bank at seven tiles a click, more than a bomb, and a
bomb held 30s was dropped on the first target in sight. So every kind is a
**charge**, kept until the player uses it: nothing lapses and nothing is used on
its own.

- **Only an account holds a charge.** `bonuses.Holder` is the account the click
  token names, and `HolderOf(clicks.PayerOf(ctx))` derives it, so the claim, the
  click chain and the drop agree on whose charge it is. A charge is the account's
  so it survives closing the tab, a new address and another device. A caller with
  no account is `NoHolder`: it holds nothing, and a scope where no account plays
  is offered nothing. Every client mints a guest account, so this leaves
  out only a caller with no token at all.
- **The rules are a value, `bonuses.Held`**: a refill and a bomb (held or not), a
  stack of enclosures, a pool of spread clicks and a stack of shields. `Granted`,
  `AfterRefill`, `AfterBomb`, `AfterEnclose`, `AfterSpreadClick` and `AfterShield`
  build a new `Held` and change
  nothing; `bonuses.Charges` writes it. `Count(kind)` is how many of a kind are held,
  and `Full(rules)` the kinds another box would add nothing to.
- **A rule set may make enclosures unlimited** (`bonuses.enclose.unlimited`): enclose then spends no charge, and
  the stack held is left as it is. See [Unlimited enclosures](enclose.md#unlimited-enclosures-bonusesencloseunlimited).
- **How much fits.** A refill and a bomb are one: a second replaces the first,
  which is what stops a stockpile of bombs being dropped all at once. Enclosures
  stack to the rule set's `bonuses.enclose.held` (3), spread clicks pool to `bonuses.spread.clicks`
  (8) and shields stack to `bonuses.shield.held` (30) (see [Rules](rules.md)). A box draws its amount evenly from 1 to `maxPerBox`, or from
  `bonuses.shield.minPerBox` for shields (`amountOf`, `crypto/rand`), and the grant caps it at the size. The claim answers
  `ClaimBonusResponse.amount` as **what was kept**: `Charges.Grant` answers the hand before and after it
  (`bonuses.Gain`), both read under the lock of its one transaction, so a player is never told of clicks that did not
  fit, and a spend in another tab cannot change the number.
- **The schedule offers no kind that is full** for any account that clicked as
  the entrant within `activeWithin` (`Registry.offerable`): the linked account
  itself, or every guest behind the scope. The schedule is by entrant and a charge
  is by account, so `bonus_click` tells the registry both on every accepted click,
  with the scope for the antibot: `Clicked(Click)`, a `Click` that names the entrant, the scope and the holder. The registry reads the charges through
  its `Holdings` port, which `bonuses.Charges` satisfies: a hand it cannot read is offered nothing.
- **Off by default, one at a time.** `ClickRequest.spread` and `enclose` say what
  the player switched on for this click, and `click_usecase.In` carries them.
  `spread_click` spends a spread click only with `Spread`, `enclose_click` an
  enclosure only with `Enclose`. **Both at once is refused** by the rule itself
  (`clicks.ErrBonusesTogether`, answered `InvalidArgument`) before anything is
  written or spent: a spread's tiles and an enclose's pocket would each take what
  the other decides. The client keeps one on at a time too, and aiming the bomb
  switches both off.
- **Another module grants with a command**, `planet.v1.GrantCharges` on `commands:planet.v1.GrantCharges`
  (`commands/apply_grant_handler`, `apply_grant_usecase`): the grant and its `grant_id` (`planet.charge_grants`) are
  kept in one transaction, so a command told twice grants once. It caps at the stack as any grant does, and answers
  nothing. `missions` pays its rewards this way. A new player is granted its refill through the same
  `charge_grants`, from player's `PlayerCreated` ([A guest's first sign-in](sign-in-refill.md)).
- **A grant reaches the player at once, on the planet stream it has open**: `charges_granted`, all it holds after the
  grant, read in the grant's transaction and sent once it is committed (`Registry.PublishGranted`, the
  `bonuses.Event`'s `Granted`), from `apply_grant_usecase` and the operator's `grant_charges_usecase`. Only a signed-in
  account's own streams hear it: a guest's stream is its address's, which others share. Delivery is at most once; the
  client reads the charges again when its stream comes back. The sign-in refill sends none: the client reads the
  charges when it signs in.
- **Nothing lapses.** Migration `20260919120000_charges_never_lapse` replaced the
  `*_until` columns with `refill` and `bomb` booleans and an `enclosures` count,
  keeping what was still in time. `bonus.chargeTTL` is gone.
- **No memory copy: every grant, spend and read goes to postgres**, one row per
  account in `planet.charges` (`refill`, `bomb`, `enclosures`, `spread_clicks`,
  `shields`). `bonuses.Charges` holds the sizes and the `Store` port, whose `Change`
  runs one transaction under `pg_advisory_xact_lock` on the account. An empty hand
  is no row. A failing database fails the call; a spread click fails with its tile
  taken. A deleted account's row stays until something removes it. The rest
  of the bonus state (schedules, offers, the hourly caps) is still memory only, so
  a restart gives everyone a fresh schedule.
- **Each spend is atomic**: `Charges.Spend` checks and takes in one `Change`, so two
  tabs racing for the last one get one, and the other `bonuses.ErrNotHeld`.
  `bonuses.StoreContractSuite` runs on postgres and on `MemoryStore` (behind the tag).
- **A box or a right answer is taken from memory before its grant**, so a grant that fails (the database down) loses
  it. That failure is not a refusal: `NotFound` is only for a token that is not the caller's or not there, and a failed
  grant goes to the error net (logged, answered `internal`). `bonus_claims_total` and `quiz_answers_total` count it
  `failed`, apart from `refused`. Once the offer is taken, the grant does not stop when the page leaves: it runs on
  `context.WithoutCancel` for at most 5 s.
- **A return undoes a spend**: `Charges.Return` puts back what a spend took, and reads no rule, since it grants
  nothing. Only a shield the map refused comes back this way (see [A shield's way](placements.md)).
- **Only a grant pushes them** (`charges_granted`, above). Otherwise they are not live news: `GetCharges` answers what the
  caller holds, read by the client at load and when its account changes, and
  `ClaimBonusResponse.charges` answers the claim, `PlaceShieldResponse.charges` a
  shield placed. After that the client follows
  its own calls: a drop spends the bomb, an accepted click sent with spread on a
  spread click, its own `tiles_enclosed` (`yours`) an enclosure. A charge spent in another
  tab shows until the next read. **`Click` answers nothing about them.** It was so
  a shadow-banned caller could not tell its clicks were dropped; a ban is told now
  (see [A ban is told](antibot.md#a-ban-is-told)), so the answer could carry them.
  `planetv1controller/chargesheld` encodes the message both procedures answer.
- **How big a charge is, is a rule, not state.** `GetBonusRules` answers the
  blast radius, the enclose's `maxTiles`, the spread pool's size and the
  enclosure stack's size, how many shields a player holds and a tile takes
  (`shields`, `tile_shields`), and every step of the toll (`toll_steps`, from
  `clicks.Toll.Steps`), all read from the rule set in force (see [Rules](rules.md)), so the client can show the
  whole table and the slowdown of a country it does not play for. It is
  `NO_SIDE_EFFECTS`, a GET no cache keeps: the client reads it at load and again on each `rules_changed`.
- **A stack held above a smaller size is kept** (`Held.Granted` never takes from a stack): a hand of 12 shields
  under a new stack of 5 keeps 12, is offered no shield box, and a grant adds none until it is under 5.
