# A guest's first sign-in earns a refill (`grant_sign_in_refill_usecase`)

**A guest that links its first identity gets one refill, once per account.** On 2026-10-28 the start bank went from
10 to 60 clicks, and newcomers still left at the moment it ran dry. The client asks a guest to sign in at that moment
(the frontend's [Sign-in](../../frontend/docs/sign-in.md)). A sign-in takes about ten seconds, which gives a guest two
clicks back at 0.2/s. The refill gives a full bank instead.

- **Auth says it, planet decides it.** `signin.Admitter` publishes `auth.v1.SignedIn` with `guest_linked` when the
  outcome is `Linked` and the account the browser was on had no identity (`accounts.Outcome.LinksAGuest`). Planet's
  `subscribers/signed_in_subscriber` (group `planet-charges-sign-ins`) hears it and runs
  `bonuses/usecases/grant_sign_in_refill_usecase`.
- **Once per account.** The grant's id comes from the account (`bonuses.SignInGrantOf`, a UUIDv5). It is kept in
  `charge_grants` in the transaction of the grant, as a `planet.v1.GrantCharges` id is. So an event told twice, or
  two tabs of one guest that link two identities at once, give one refill. There is no unlink: an account stops being
  a guest once.
- **No other sign-in earns it.** An identity that already has an account moves the browser there (`SignedIn`). A
  verified address that another account holds joins that account (`Joined`). A new account is `Created`. A second
  provider on a linked account is `Linked`, but the account is no guest. A sign-out then a sign-in is `SignedIn`. The
  guest the browser leaves gets nothing. So a switch of account earns nothing, and each refill needs a new identity:
  a Google or Discord account, or an email address that is not disposable ([Auth](auth.md)).
- **What farming wins.** The refill fills what the player spent from its own bank. A guest spends from its network's
  shared guest bank too ([Rate limiting](rate-limiting.md)), so all the refills that a network's guests can earn are
  bounded by what that shared bank let through: 0.2 clicks a second and one bank. A new account already starts with
  `clicks.newAccountClicks`.
- **It is the refill, nothing new.** The grant is a refill charge, and a player holds one at most: a guest that
  already holds one gets nothing more. It is spent through `UseRefill` ([Refill](refill.md)) with the click token
  that the client mints after the sign-in. That token is linked, so the refill fills the account's own bank and its
  scope's, never the guests' shared bank.
- **The client spends it at once**, so the meter shows a full bank when the guest is back.
- **A rule: `bonuses.signInRefill`** ([Rules](rules.md)), on in every set since migration
  `20261030120000_sign_in_refill`. Off, a first sign-in earns nothing, and `GetBonusRules.sign_in_refill` tells the
  client, which then promises no full bank. The rule in force when planet hears the event decides.
- **A new group reads its stream from the start** ([The composite layer](composite-layer.md)). At its first deploy
  planet read every earlier sign-in. None of them had `guest_linked`, so none gave a refill.
