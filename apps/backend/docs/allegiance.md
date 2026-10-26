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
- **`allegiances.Book.Pledge(ctx, payer, flag)`** applies `Pledged` at the clock's
  time, in the store's `Change`: one transaction under `pg_advisory_xact_lock` on the
  account, so two tabs pledging two flags at once cannot both change it. A caller with
  no account is `ErrNoAccount`, and a flag that is no country `clicks.ErrUnknownCountry`.
  **`Book.PlayError(ctx, payer, flag)`** reads the store and asks `PlayError`. A caller
  with no account (the deprecated mint, `auth` off) is held to nothing, and a flag that
  is no country is left to the click use case, which refuses it.
- **It is checked after the ban and before the throttle** (`allegiance_click`,
  between `antibot_attempt_click` and `throttle_click`): a banned caller is told its
  ban first, and a click refused for its flag spends no token.
  `place_shield_usecase` checks before it spends the shield. A bomb, a claim, a
  quiz and a refill name a flag only for what they announce, and are not held.
- **The refusal is `FailedPrecondition` with a `planet.v1.Allegiance` detail**
  (`allegiancemessage.Refusal`): the flag held and `changeable_at_unix_ms`. The
  client snaps back to that flag. With no flag held, the detail has no country.
- **`ClickService/GetAllegiance`** answers the caller's (`allegiance_query`, straight
  from postgres), as `Identified`: unset for an account that never pledged a flag,
  and `changeable_at_unix_ms` 0 for a flag never changed. Not `NO_SIDE_EFFECTS`.
- **`InternalService/GetAllegianceOf(account_id)`** is the same read for another module
  (`missions`, before the day's first take), with the same query. An id that is no account is `InvalidArgument`.
- **`ClickService/PledgeAllegiance(country_id)`** is `Book.Pledge`, and answers
  nothing: the client reads `GetAllegiance` after it. `Identified`, since it writes
  only the caller's own state. No account is `Unauthenticated`, a flag that is no
  country `InvalidArgument`, and a change inside the hour is refused as above. **The
  web client pledges** when the player picks a country, and before the first click of
  an account that holds no flag: a new guest gets its account from the mint of that
  click, so it has none at load.
- **No memory copy: every click and every shield reads postgres**, and a change writes
  it, in `planet.allegiances` (`account`, `flag`, `changed_at`, NULL for a first flag).
  Only the tile map is kept in memory and flushed. A failing database fails the click,
  as the error net's `internal`. `allegiances.StoreContractSuite` runs on postgres and
  on `MemoryStore`, the fake the use cases are tested over (behind the tag).
- **A row is never deleted**, not even with its account: without the row the next flag
  is a first flag again, which would hand out a free change. The row names nobody once
  the account is gone.
- **Guests are held per account**, like the bank: a private tab is a new guest and
  a new first flag.
