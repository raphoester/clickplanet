# The flag played for

**The server holds each account to one flag for an hour after a change** (the
backend's [Allegiance](../../backend/docs/allegiance.md)). **Only `PledgeAllegiance` sets or changes it**: a
click or a shield for another flag than the one pledged is refused, locked or not,
and so is one from an account that pledged none.

- `AllegianceSource` (`backends/backend.ts`) is the flag held, `Allegiance` in
  `domain/allegiance.ts`: the country and `changeableAt`, undefined for a flag
  never changed. `lockOf` is whether it holds now.
- **`pledgeAllegiance(country)` changes it** when the player picks a country:
  `PledgeAllegiance` with the identity token, then `GetAllegiance`, since the pledge
  answers nothing. A refusal is `AllegianceLockedError`. A visitor with no account
  pledges nothing: its first click will.
- **The client pledges before the first click of an account with no flag**
  (`ensureFlag` in `PlanetBackend`): once per click token, it reads the flag held
  for that token, and pledges the flag clicked for when there is none. A new guest
  gets its account from the mint of its first click, so it has none at load; a
  sign-in or a sign-out changes the account the same way. Clicks sent at once share
  one read. A pledge that failed sends no click, and the next click tries again. A
  token whose flag was read at load needs no read.
- **`PlanetBackend` reads `GetAllegiance`** at load with the identity token, and
  again when a click goes out under a new token (`followSession`). A refusal carries
  the flag held in its detail, which `PlanetBackend` keeps (`allegianceDetailOf`).
- **The picked flag is always the held one**, locked or not: `Viewer` sets the
  country back to it (local storage, a share link, a CrazyGames invite, another
  device). `Viewer`'s
  `pickCountry` pledges first and sets the country only once the server took it.
- **The picker says the rule over the list** (`CHANGE_RULE`, "You can change
  country once an hour."), and while the flag is held, when the next change is,
  with no list. **There is no confirm step**: a tap on a country switches, since
  the rule is right above it. A pick that failed for another reason says so, and
  the picker stays open.
- **The Change button shows the lock**: a padlock and the minutes left
  (`minutesLeftOf`), named with the time of the next change.
- **A refused click is rolled back, plays the refusal, and says why** in a bubble
  over the clicks meter, "You play for Germany until 16:47" (`heldLine`, with no
  time when no wait runs); a refused shield says the same. It only happens on a
  page that did not know the flag changed elsewhere.
- `FakeBackend` holds the flag the same way, and takes a first flag on the first
  click, as the client's pledge would.
