# A newcomer's first box and first quiz (`bonuses/welcome.go`)

**An account made less than the offers' rule set's `welcome.within` ago (10m) gets its first box and its
first quiz early, once each.** Measured from 2026-10-02 to 10-22: 36% of new accounts left
inside two minutes, most at the 10th or 11th click (`clicks.newAccountClicks`), and
the first box and quiz came 4-8 minutes in, so they never saw one.

- **The first box is a spread, `welcome.boxAfter` (1m) after the account was made.**
  The first click of a young account pulls its entrant's next box to then, if that is sooner,
  and owes it the spread. The age is the account id's (`Click.Born`, which the offers' `click_tried_subscriber` reads
  off the UUIDv7 id `ClickTried` names, `cpsession.AccountID.CreatedAt`), so it counts from the account, not from the
  click, and a restart does not move it.
- **The first quiz comes the moment a young account's click empties its bank**, and a right
  answer is shields. Planet's `try_click` tells the bank the click left as `ClickTried.dry`
  (`out.Limited && out.Budget.Empty()`, fewer than one click, which is when the client stops sending), and the offers
  read `Click.Dry` off it. The quiz is due when that click is heard, about 0.1 s after it was tried, and the next sweep
  (1s) asks it: at most about 0.1 s later than until 2026-11-11, when `bonus_click`, outside `throttle_click`, read the
  bank inside planet's click chain. A `ClickTried` Redis refused is a dry click never heard: the early quiz waits for
  the next. While quizzes are off (`quiz.enabled`), none is owed.
- **Never a refill or a bomb.** `newAccountClicks` exists because one phone made an account a
  minute for a fresh bank; a welcome refill would hand that back, and a bomb would be a free strike
  per account. A spread click still costs a click, and a shield is placed with none, so a
  player with an empty bank can still use it. A guest gets its one refill when it signs in
  ([A guest's first sign-in](sign-in-refill.md)): that needs a new identity, not a new account. **A welcome kind that cannot be
  offered** (weight 0 in `kinds`, or a full stack) **gives no early offer**: the turn goes
  back to the ordinary window, never to another kind.
- **A full stack is the wallet's to say**: the welcome kind is left out when planet's room (`GetRoom`) says the account holds
  as many as it may ([Offers and the wallet](offers-and-wallet.md#room)). An account whose first click lands while
  the room is asked waits for the next sweep, so its welcome is not lost to that race.
- **Once per account**: `Registry.newcomers` remembers each young account until it is no longer
  young, so neither a second click nor a second empty bank pulls anything forward again. It is
  memory only, so a restart can welcome an account a second time while it is still young.
- **A welcome waits only for the other to be gone.** A box and a quiz keep `spacing` apart (see
  [Quizzes](quizzes.md)), but a welcome is not held for it, since its moment is the point: it is held only while the
  other is out, at most a box's 15s flight or a banner's 25s and its 8s. An ordinary box or quiz beside a welcome is
  held for the spacing as any is.
- Everything else is the ordinary offer: the stream must be open, the entrant active, the
  hourly caps not reached, and the guests behind a scope share one entrant, so a private tab a
  minute buys no more than the caps.
- **Off with `within` 0s.** An account welcomed keeps its welcome when the rule changes; a new rule applies
  to the accounts it meets after.
