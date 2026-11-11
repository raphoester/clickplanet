# A guest's first sign-in earns a refill (`grant_sign_in_refill_usecase`)

**A new player gets one refill, once per account.** An account becomes a player when `player` makes its profile,
which happens at its first sign-in. On 2026-10-28 the start bank went from 10 to 60 clicks, and newcomers still left
at the moment it ran dry. The client asks a guest to sign in at that moment (the frontend's
[Sign-in](../../frontend/docs/sign-in.md)). A sign-in takes about ten seconds, which gives a guest two clicks back at
0.2/s. The refill gives a full bank instead.

- **The offers hear player, not auth.** `player` publishes `player.v1.PlayerCreated` with the profile it makes
  ([Player](player.md)). The bonuses module's `subscribers/player_created_subscriber` hears it under group
  `planet-charges-players`, planet's name, kept when the offers left planet (2026-11-11) so the group goes on where
  planet's subscriber left the stream, and runs `bonuses/usecases/grant_sign_in_refill_usecase`. So the game reads no
  sign-in of auth's for this. The refill is a granting rule, so it is the offers'; it pays through planet's
  `InternalService/GrantCharges`, with `lasting` ([The offers](bonuses.md#the-sign-in-refill)).
- **Once per account.** The grant's id comes from the account (`bonuses.SignInGrantOf`, a UUIDv5, the same id for an
  account as before the cut and the move, pinned by a test, so a refill planet granted before is not granted again).
  Planet's payment (`charges/usecases/pay_usecase`, behind `GrantCharges` with `lasting`) keeps it in `charge_grants`
  in the transaction of the grant, as a `planet.v1.GrantCharges` command's id is. So an event told twice gives
  one refill. `player` makes a profile once per account.
- **Who is a new player.** A guest that links its first identity keeps its account, and `player` names it: it earns
  the refill. An account made at sign-in (a browser with no guest) is a new player too. An identity that already has
  an account moves the browser there, and a verified address that another account holds joins that account: that
  account had its profile, so nothing is earned, and the guest the browser leaves gets nothing. A second provider, or
  a sign-out then a sign-in, makes no profile either. So a switch of account earns nothing, and each refill needs a
  new identity: a Google or Discord account, or an email address that is not disposable ([Auth](auth.md)).
- **What farming wins.** The refill fills what the player spent from its own bank. A guest spends from its network's
  shared guest bank too ([Rate limiting](rate-limiting.md)), so all the refills that a network's guests can earn are
  bounded by what that shared bank let through: one bank, refilled at 0.2 clicks a second, or at the signed-in pace
  while a guest in its fast start clicks. An account made at sign-in already
  starts with `clicks.newAccountClicks`, so its refill is one more bank for one more identity.
- **It is the refill, nothing new.** The grant is a refill charge, and a player holds one at most: a guest that
  already holds one gets nothing more. It is spent through `UseRefill` ([Refill](refill.md)) with the click token
  that the client mints after the sign-in. That token is linked, so the refill fills the account's own bank and its
  scope's, never the guests' shared bank.
- **The client spends it at once** after a guest's first link, so the meter shows a full bank when the guest is
  back. An account made at sign-in has a full bank already: its refill waits in the inventory.
- **A rule: the offers' `signInRefill`** ([Its rule sets](bonuses.md#its-rule-sets)), on in every set since planet's
  migration `20261030120000_sign_in_refill`, and copied into the offers' sets on 2026-11-11. Off, a new player earns
  nothing, and `bonuses.v1.BonusService/GetGrantRules` (`sign_in_refill`; `GetBonusRules` until 2026-11-11) tells the
  client, which then promises no full bank. The rule in force when the offers hear the event decides.
