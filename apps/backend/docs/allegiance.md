# Allegiance: one change of flag an hour

**An account plays for one flag, and may change it once an hour.** Every click and
every shield names a flag, and nothing else ties a player to one: before this, a
player could paint for one country and defend another click to click. **Only
`PledgeAllegiance` sets or changes the flag: a click or a shield never does.**

- **The rules are on `Allegiance`**, pure and tested in the `allegiances` root.
  `Pledged(flag, at)` is a pledge: the first flag is kept with no wait, the next
  change is free too and starts the wait, and another flag within `changeEvery` (1h)
  of it is `allegiances.Refusal` (`ErrChangedTooSoon`), with the flag held and until
  when. The same flag again changes nothing. `PlayError(flag)` is a click or a
  shield: the flag held plays, any other flag is a `Refusal` (`ErrOtherFlag`) whether
  a wait runs or not, and an account with no flag yet is one (`ErrNoFlag`).
- **A rule set may lock the flags** (`allegianceLocked`, part `allegiances.LockPart`, see [Rules](rules.md)): a
  battle's set does, so no player switches sides in the middle of a fight (the owner, 2026-10-31). Planet never hears
  of a battle: the lock is a rule of the set in force, like a freeze. While it is on, `Pledged(flag, at, lock)` refuses
  another flag to an account that holds one (`ErrLocked`), even after the hour, and the refusal says `Locked`. A first
  flag and the same flag still go, so a newcomer joins mid-battle. The hour's wait is not touched: a change before the
  battle still counts after it. `PlayError(flag, lock)` says the lock in its refusal too, so a page that hears a refused
  click learns it.
- **While the knock-out plays, only a flag still in may be picked** (the owner, 2026-11-09): `Pledged(flag, at, lock, standing)` reads
  `Standing.Stands`, which the tile storage answers (see [The knock-out](capitals.md#the-knock-out-capitalsknockout)). A flag knocked
  out, or with no capital (England, Scotland, Wales, Brittany), is refused, `ErrDown` (`FailedPrecondition`), to a
  newcomer too. **An account whose flag does not stand changes at once, past the lock and the hour**: the change
  starts the hour's wait as any change does. While the knock-out does not play every flag stands, so nothing changes
  there. After the battle everybody keeps the flag picked. `PlayError` is not touched: with the reach, a flag with no
  land reaches nothing already (without it, see [Arena battles](arenas.md#known-gaps)).
- **`allegiances.Book.Pledge(ctx, payer, flag)`** applies `Pledged` at the clock's
  time, in the store's `Change`: one transaction under `pg_advisory_xact_lock` on the
  account, so two tabs pledging two flags at once cannot both change it. A caller with
  no account is `ErrNoAccount`, and a flag that is no country `clicks.ErrUnknownCountry`.
  **A pledge that sets or changes the flag is told**, once written: `Teller.Pledged`, which
  `allegiance_facts` tells as `planet.v1.AllegiancePledged` (the account, the flag, the
  flag it left, empty for a first one) through the facts log, as the other facts go. The
  same flag again, a refusal and a release tell nothing. `alerts` hears it to move the
  player's Discord role (see [Discord pings](alerts.md)), and `chat` to end the player's stream of the room of the
  flag it left (see [Chat](chat.md#rooms)).
  **`Book.PlayError(ctx, payer, flag)`** reads the store, with 1s to do it, and asks `PlayError`. A caller
  with no account (`auth` off) is held to nothing, and a flag that
  is no country is left to the click use case, which refuses it.
  **A flag that cannot be read refuses nothing**: the read's error is `ErrUnread`, which `allegiance_click` and the
  shield's gate let through and `log_allegiances` logs, as a refusal that cannot be read does (see
  [Anti-bot](antibot.md)). Postgres down lets an account play a flag it did not pledge, rather than stop every click;
  a shield still needs postgres to be spent.
- **It is checked after the refusal and before the throttle** (`allegiance_click`,
  between `refusal_click` and `throttle_click`): a refused caller is told its
  refusal first, and a click refused for its flag spends no token.
  `place_shield_usecase` checks before it spends the shield. A bomb, a claim, a
  quiz and a refill name a flag only for what they announce, and are not held.
- **The refusal is `FailedPrecondition` with a `planet.v1.Allegiance` detail**
  (`allegiancemessage.Refusal`): the flag held, `changeable_at_unix_ms` and `locked`. The
  client snaps back to that flag. With no flag held, the detail has no country.
- **The operator may lift the wait** (`AdminService/ReleaseAllegiance`, see [Operator tools](admin.md)):
  `Released()` keeps the flag and forgets the change, so the next change is free as the first one is. A lock still
  holds.
- **`ClickService/GetAllegiance`** answers the caller's (`allegiance_query`, straight
  from postgres, and `locked` from the set in force), as `Identified`: unset for an account that never pledged a flag,
  and `changeable_at_unix_ms` 0 for a flag never changed. Not `NO_SIDE_EFFECTS`. A set taking over sends
  `rules_changed`, and the web client reads it again.
- **`InternalService/GetAllegianceOf(account_id)`** is the same read for another module
  (`missions`, to draw an account's day; `chat`, to let an account into its flag's room), with the same query. An id that is no account is `InvalidArgument`.
- **`ClickService/PledgeAllegiance(country_id)`** is `Book.Pledge`, and answers
  nothing: the client reads `GetAllegiance` after it. `Identified`, since it writes
  only the caller's own state. No account is `Unauthenticated`, a flag that is no
  country `InvalidArgument`, and a change inside the hour or under a lock is refused as above. **The
  web client pledges** when the player picks a country, and before the first click of
  an account that holds no flag: a new guest gets its account from the mint of that
  click, so it has none at load.
- **No memory copy: every click and every shield reads postgres**, and a change writes
  it, in `planet.allegiances` (`account`, `flag`, `changed_at`, NULL for a first flag).
  Only the tile map is kept in memory and flushed. A failing database lets the click
  through, as above. `allegiances.StoreContractSuite` runs on postgres and
  on `MemoryStore`, the fake the use cases are tested over (behind the tag).
- **A row is never deleted**, not even with its account: without the row the next flag
  is a first flag again, which would hand out a free change. The row names nobody once
  the account is gone.
- **Guests are held per account**, like the bank: a private tab is a new guest and
  a new first flag.
