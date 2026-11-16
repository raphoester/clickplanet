# The rally

**A rally is a charge that calls the player's flag to one place** (the design "Player Level and Unlocks", the owner,
2026-11-16). The player taps Rally, then a point of the globe. A ring appears there, about the size of France:
`charges.rally.rings` tile spacings wide (18, about 1,000 tiles). Every player sees it. For `charges.rally.lasts`
(3 minutes), a click of the rally's flag on a tile inside the ring takes `charges.rally.cost` (half a click) from the
bank instead of one. The chat tells the flag's room: "le professeur calls a rally in Tasmania".

**It is built and dormant**: no box and no right answer is a rally yet, as the bonuses' `kinds.rally` weighs 0 in
`default` and `gates.open.rally` is off there (migration `20261116152259_rally_offers`), and no other set holds either.
`unlocked_kind` has `rally` (`20261116152258_unlocked_rally`), and level 10 unlocks it ([Levels](levels.md#the-unlocks)).
The operator's `GrantCharges` (`rallies`) gives some, and so does the fake on the page.

## The rules

- **A hand holds 3 rallies at most** (`charges.rally.held`), a stack as mines: a grant caps at it. A box gives one once it
  weighs above 0 (the bonuses' `rally.maxPerBox`, 1).
- **The ring is a circle on the globe around a point, not a tile**: every tile whose centre is within
  `charges.rally.rings` × the map's tile spacing of the arc of the point tapped (1 to 50, 18 in `default`). It is not a
  walk over neighbours: it covers an island just offshore. **Why a point**: the area is what counts, and no tile at its
  centre gets anything the others do not (the owner, 2026-11-16). So the ring sits where the finger was, and a ring on
  the sea that reaches a coast is a ring.
- **It lasts `charges.rally.lasts`** (10 s to 30 min, 3 min in `default`).
- **A click inside a ring of its flag takes `charges.rally.cost` from the bank** (0.1 to 1, 0.5 in `default`), from every
  bucket the click spends from: the account's, the network's guests' and the network's.
- **Rings of one flag that overlap cost no less**: half a click, never a quarter. Rings of two flags: each flag's clicks
  get only their own, so where they meet is a battlefield.
- **A player has one rally on at a time**: a call while its last is on is refused, and the rally stays held.

The owner's answers (2026-11-16):

- **Every player sees the ring**, the enemy too: it is on every stream and in a public read.
- **The room line names the landmass**, as the bomb's line does. The design said "near Lyon", and the server knows no
  cities.
- **A hand holds a stack of 3**, as mines.
- **A cheaper click, not a faster refill.** The design doubled the bank's pace for a take inside the ring. A pace is set
  by each take and lasts until the next one (see [Rate limiting](rate-limiting.md)), so a player whose last click was
  inside would refill twice as fast after the ring ended. The bank was already a number with a fraction, on the server,
  on the wire and on the page, so a click costs half instead: the effect is exactly the ring and its time. **It is
  stronger**: a full bank of 60 buys 120 clicks inside, so a signed-in player alone gets about 132 more clicks in 3
  minutes, against about 72 with a doubled pace. `charges.rally.cost` is the knob.

Chosen where the design is silent:

- **Any point of the globe takes a rally**, on land or sea, whoever holds the tiles under it. A rally to attack or to
  defend. A call is no take, so the reach does not hold it.
- **A ring belongs to the flag it was called for.** A caller who pledges another flag leaves it to the old flag's
  players, and calls no other until it ends.
- **A ring keeps the radius and the end it was called with**: a change of the rules reaches the next one. The cost is
  read at each click.
- **A ring changes the price of a click, never what the click does**: a shield still takes the click, a capital is still
  a capital, and the reach still holds. A click inside needs only half a click in the bank.

## The call (`CallRally`)

`ClickService/CallRally{point, country_id}`, session-gated and frozen like `LayMine`, and not throttled: holding a
rally the server granted is the gate. `point` is a `GlobePoint`, as the bomb's: any vector of some length, put on the unit
globe (`rallies.PointOf`); none, a zero or a NaN is `InvalidArgument` (`rallies.ErrNoPoint`) before anything is read.
`rallies.Gate` refuses before the rally leaves the hand, in this order: a banned caller (`BanRefusal`), an unknown flag
(`InvalidArgument`), a flag the caller does not play for (the allegiance's refusal), and a rally of the caller still on
(`RallyRefusal`, `rallies.ErrRallyOn`). A failed read of the refusals, the flag or the last rally refuses nothing, as for
a click.

Then `call_rally_usecase` spends one rally (`Held.AfterRally`; none held, or no account, is `NotFound`), makes it by the
rules in force (`rallies.Muster`: an id, the ring's radius, its end) and keeps it as the account's last under a lock on
the account (`Store.Change`, `Rally.Called`), which refuses it again if a second tab called first: the rally spent comes
back to the hand (`Held.WithRallyBack`, with no cap). The answer is the rally with all its time left.

**Each rally called is told twice** (`rallies.Tellers`): to every stream open (`rally_streams`, below) and to the other
modules as `planet.v1.RallyCalled` (`rally_facts`, through planet's facts log), with its point, and the ground and the
landmass of the tile nearest it (`Geography.Nearest`, read off `clicks.Borders`) when that tile is inside the ring. A ring
with no tile inside names none, and the room line names no place.

## The price of a click

`throttle_click` asks the `rallies.Discount` what the click costs before it spends: it reads the rallies of the flag
clicked for that are still on (`Store.On`, one indexed read of postgres per click, as the allegiance's, with a second to
answer), and answers `charges.rally.cost` when one of them covers the tile, a whole click otherwise. It spends that from
every bucket (`Limiter.TakeAll(cost, …)`), and the answer says so: `ClickBudget.spent`. **A failed read costs a whole
click**, and `log_discount` logs it: postgres down takes the discount away, never a click.

`GetBudget` and `UseRefill` read no rally: a reading has no tile, and a refill fills a bank whatever it costs.

**The antibot reads a click inside a ring at the tempo its cost makes**: `ClickTried.tempo` is the tempo in force over
what the click took, so half a click at the plain tempo reads 2. The metronome reads each gap times the tempo (see
[Anti-bot](antibot.md)), so a player clicking twice as fast inside a ring reads as at its bank's pace, as in a battle at
2×.

## Who sees it

**Every player**, whatever its flag:

- **`rally_called`** on every planet stream (`notices.Board.PublishRally`), with `left_ms`, the time it has left as it
  is told.
- **`ClickService/GetRallies`** answers every rally still on, each with its `left_ms` now, the soonest to end first
  (`rallies_query`, from postgres). Public, no session. Not `NO_SIDE_EFFECTS`: the time left changes each read, and a
  cached answer lies. The page reads it as its stream opens.
- **`GetMap` holds none**: a rally changes no tile, so the map's version and its ETag do not move.
- **`GetChargeRules`** answers `rallies` (the stack), `rally_radius` (in radians, for the aim), `rally_lasts_ms` and
  `rally_cost`.

## The line in the flag's room

Chat hears `planet.v1.RallyCalled` (`rally_called_subscriber`, group `chat-announcements-rallies`) and
`announce_rally_usecase` names the caller as player names it now, then keeps a `rally` announcement
`{name, country, ground, landmass}` in the flag's room (the world shows every one, see
[Announcements](chat.md#announcements)), its id the rally's, so a fact told twice is one line. Migration
`20261116152300_rally_announcements` adds the kind. **Planet's facts log is at most once**: a restart in the tenth of a
second before its relay loses the line, never the rally.

## A restart

**A rally is a row in postgres, not map state** (`planet.rallies`: the account's last rally, on or ended, one row per
account, upserted by each call; its centre in `x`, `y`, `z`). Nothing of it is in memory, so a restart loses nothing.

## Arenas, resets and remaps

- **A ring is a point on the globe and a time**, not a tile's state: it lasts through an arena's open and close, a
  reset and a remap, which keep the same globe, until its end. `scripts/map/remap.mjs` leaves `rallies` alone.
- **A freeze refuses a call** (`FrozenRefusal`). A ring called before goes on, and no click is taken while frozen.
- **A knock-out leaves a ring of the flag out**: its players must pick another flag, and the ring helps no click of it.

## The ledger: no kind, and nothing new reaches `TilesTaken`

**A rally takes no tile.** The takes inside the ring are the clicks the players make, with their own kinds (`click`,
`spread`, `enclose`), and they reach `TilesTaken` as any take: the stats, the XP, the missions and scoring count them for
whoever clicked. **Why it differs from a virus and a mine**: those take tiles with a charge, so their takes need a kind
of their own to stay in the ledger's parity. A rally changes the price of the players' own clicks and nothing of the
map, so the ledger, the map's history, has nothing to keep. The call itself is kept as the account's last rally and
told as a fact, as a mine's lay writes no ledger event either.

## Known gaps

- **A banned player's rally goes on**, for its flag's other players.
- **A deleted account's row stays**: it names nobody once the purge forgot the account, as an allegiance's row.
- **The room line is at most once**, as above.

## Tests

`rallies`' `rally_test.go` (on, one at a time, the ring around a point between tiles, `PointOf`), `muster_test.go` (the
rules as it is called), `gate_test.go` (any point, the flag, a ban, one still on, the failed reads), `discount_test.go`
(inside, outside, another flag, rings that overlap, a ring ended, the cost read at each click, a failed read) and the
store's contract on memory and on postgres; `call_rally_usecase` (the spend, the rally kept and told once, the gate's
refusal, a second tab, none held, no account, a store down, no point). `throttle_click`'s cost and `try_click`'s tempo;
`notices`' board (a rally to every stream); the sink; `call_rally_handler`'s codes; `rallies_query`; `rally_facts` (the
landmass inside the ring, none beyond); `GetChargeRules`. The planet rules' bounds, and the bonuses' rules pin that
`default` weighs the rally 0 and gives one. Chat's announcement, `announce_rally_usecase` and the subscriber.
