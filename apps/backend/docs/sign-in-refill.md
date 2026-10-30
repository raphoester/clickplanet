# A guest's first sign-in earns a refill (`grant_sign_in_refill_usecase`)

**A new player gets one refill, once per account.** An account becomes a player when `player` makes its profile,
which happens at its first sign-in. On 2026-10-28 the start bank went from 10 to 60 clicks, and newcomers still left
at the moment it ran dry. The client asks a guest to sign in at that moment (the frontend's
[Sign-in](../../frontend/docs/sign-in.md)). A sign-in takes about ten seconds, which gives a guest two clicks back at
0.2/s. The refill gives a full bank instead.

- **Planet hears player, not auth.** `player` publishes `player.v1.PlayerCreated` with the profile it makes
  ([Player](player.md)). Planet's `subscribers/player_created_subscriber` (group `planet-charges-players`) hears it and
  runs `bonuses/usecases/grant_sign_in_refill_usecase`. So the game reads no sign-in of auth's for this.
- **Once per account.** The grant's id comes from the account (`bonuses.SignInGrantOf`, a UUIDv5). It is kept in
  `charge_grants` in the transaction of the grant, as a `planet.v1.GrantCharges` id is. So an event told twice gives
  one refill. `player` makes a profile once per account.
- **Who is a new player.** A guest that links its first identity keeps its account, and `player` names it: it earns
  the refill. An account made at sign-in (a browser with no guest) is a new player too. An identity that already has
  an account moves the browser there, and a verified address that another account holds joins that account: that
  account had its profile, so nothing is earned, and the guest the browser leaves gets nothing. A second provider, or
  a sign-out then a sign-in, makes no profile either. So a switch of account earns nothing, and each refill needs a
  new identity: a Google or Discord account, or an email address that is not disposable ([Auth](auth.md)).
- **What farming wins.** The refill fills what the player spent from its own bank. A guest spends from its network's
  shared guest bank too ([Rate limiting](rate-limiting.md)), so all the refills that a network's guests can earn are
  bounded by what that shared bank let through: 0.2 clicks a second and one bank. An account made at sign-in already
  starts with `clicks.newAccountClicks`, so its refill is one more bank for one more identity.
- **It is the refill, nothing new.** The grant is a refill charge, and a player holds one at most: a guest that
  already holds one gets nothing more. It is spent through `UseRefill` ([Refill](refill.md)) with the click token
  that the client mints after the sign-in. That token is linked, so the refill fills the account's own bank and its
  scope's, never the guests' shared bank.
- **The client spends it at once** after a guest's first link, so the meter shows a full bank when the guest is
  back. An account made at sign-in has a full bank already: its refill waits in the inventory.
- **A rule: `bonuses.signInRefill`** ([Rules](rules.md)), on in every set since migration
  `20261030120000_sign_in_refill`. Off, a new player earns nothing, and `GetBonusRules.sign_in_refill` tells the
  client, which then promises no full bank. The rule in force when planet hears the event decides.
