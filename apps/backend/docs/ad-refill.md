# A rewarded ad fills the bank (`claim_ad_refill_usecase`)

**On CrazyGames, a player at an empty bank may watch a rewarded ad, and the server fills the bank.** `ClaimAdRefill`
fills it. `GetAdRefill` says when the caller's bank may have the next one.

- **The server never sees the ad.** CrazyGames gives no receipt of a watched ad that a server can check, so a client
  that lies can call `ClaimAdRefill` without one. The server bounds what that wins: one full bank per bank in each
  span of `bonuses.adRefill.every` (1 h), never past a full bank, and only with a click token (session-gated like
  `UseRefill`, see [Sessions](sessions.md)). A liar gets what an honest player who watches one ad an hour gets.
- **It is a refill, not a charge.** No charge is granted or spent. It fills the bank as a refill does
  ([Refill](refill.md)): `clicks.Refilling` fills the bank and gives the scope's bucket what that gave.
  `use_refill_usecase` and `claim_ad_refill_usecase` share it.
- **The spans.** A span starts at a multiple of `every` from the Unix epoch: with 1 h, on each hour, UTC. Every bank
  has the same spans. So a player who watched at 12:59 may watch again at 13:00: two banks close together, then one an
  hour.
- **One per bank.** `bonuses.AdRefills` makes the grant ids, UUIDv5 like the sign-in refill's: one per account and
  span. A guest also has one per network (`clicks.Payer.Scope`) and span, because the guests behind one network spend
  from one bank (`guests:<scope>`, see [Rate limiting](rate-limiting.md)). So the guest tabs and the guests of a
  household get one ad refill in a span between them. A signed-in player has only its own: it does not spend from the
  guests' bank, so it neither takes the network's nor is refused for it. A guest that signs in keeps the one its
  account had.
- **Kept in `charge_grants`**, as `GrantCharges` and the sign-in refill keep theirs (`postgres_grant_store`: `Record`
  inserts, `Recorded` reads). A claim records all its ids in one transaction: a second claim in the same span waits on
  the first one's row, then finds it and loses, and a refused claim keeps none of its ids. The span's start names it,
  so a new `every` gives no second refill in a span that starts with the old one.
- **What `ClaimAdRefill` refuses, in order:**
  - the rule is off: `FailedPrecondition` (`claim_ad_refill_usecase.ErrAdRefillOff`);
  - no account: `NotFound` (`ErrNoAccount`);
  - a full bank: `FailedPrecondition` (`clicks.ErrBankFull`), and nothing is recorded, so an ad watched at a full
    meter wastes nothing;
  - a bank that had its ad refill in this span: `FailedPrecondition` with a `planet.v1.AdRefillRefusal` detail,
    `available_at_unix_ms` the start of the next span (`bonuses.AdRefillRefusal`).
  A frozen map refuses it before all of these, as it refuses `UseRefill` ([A frozen map](rules.md#a-frozen-map)).
- **The answer is the budget**, full, so the meter jumps at once.
- **`GetAdRefill` needs only an identity token**, as `GetCharges` does. It answers when the caller's bank may have the
  next one: the start of the span now when none of its ids is recorded, else the start of the next span. At or before
  now, it may now. A caller with no account is answered 0. It does not read the rule's switch:
  `GetBonusRules.ad_refill` tells the client whether it is on.
- **A rule: `bonuses.adRefill`** ([Rules](rules.md)): `enabled` and `every`, on and 1 h in every set since migration
  `20261030180000_ad_refill`. `every` must be above 0.
