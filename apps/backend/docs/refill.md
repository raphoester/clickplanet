# What a refill does to the bucket

`UseRefill` spends it, through `use_refill_usecase`. A refill comes from a box, a
mission, or a guest's first sign-in ([A guest's first sign-in](sign-in-refill.md)). A rewarded ad fills the bank the
same way, with no charge ([A rewarded ad](ad-refill.md)). **A refill is a full bank
the player can spend**: what it gives is what the player lacks of one, its own
bank's size less the emptiest bucket it spends from (`lack`). `clicks.Refilling` holds that rule for both: `Full`,
`Refill`, and the `Budget` the answer carries.

- **It fills the bank** (`Buckets.Bank`): the account's bucket and, for a guest,
  its scope's guests', topped up to capacity by `cpratelimit.Limiter.Fill(key)`.
  The guests' is filled because the guests on one network are one bank: a
  refill that left it empty would be a refill of nothing. A guest's boxes are
  already offered per scope, so this is no new allowance.
- **It gives the scope's bucket what it gave, never more** (`Buckets.Ceiling`,
  `Limiter.Add(key, lack)`), and never past its capacity. The scope's is shared
  with every other player behind the address, so filling it would be a refill
  for all of them; leaving it alone made the refill drain it instead. Two
  signed-in players on one IPv6 line exactly fill a /64's 4× at their pace, so
  each refill left the scope one bank lower than the banks behind it, until it
  was the tighter bucket: the meter read the scope, the refill filled a bank
  that was already full, and was refused as one (2026-10-23). Given what the
  refill gave, the scope is no tighter for anyone else, and the refilled player
  can spend all of it. `TestARefillGivesABankWhenTheNetworksBucketIsTheTighter`
  replays that household over HTTP.
- The bank never grows past its size, and the pace is left as it was. `Fill`
  and `Add` are **additive**: nothing that never calls them can tell they exist,
  which matters because the same limiter type throttles chat and session mints.
- **A full bank is refused before the charge is touched**: `clicks.ErrBankFull`,
  answered `FailedPrecondition`, spends nothing. Full is a `lack` of 0: the bank
  full and the scope holding at least a bank, which is the reading the client
  sees full. The client checks first and says "Bank already full" without
  asking; the server check is what holds.
- **No refill, or no account**: `ErrNoRefill`, answered `NotFound`.
- **The answer carries the budget**, full, and the charges, so the meter jumps at
  once rather than on the next click.
- `UseRefill` is session-gated like `Click` and `DropBomb`, and not throttled:
  holding the refill the server granted is the gate.
