# A newcomer's first box and first quiz (`bonuses/welcome.go`)

**An account made less than the rule set's `bonuses.welcome.within` ago (10m) gets its first box and its
first quiz early, once each.** Measured from 2026-10-02 to 10-22: 36% of new accounts left
inside two minutes, most at the 10th or 11th click (`clicks.newAccountClicks`), and
the first box and quiz came 4-8 minutes in, so they never saw one.

- **The first box is a spread, `bonuses.welcome.boxAfter` (1m) after the account was made.**
  The first click of a young account pulls its entrant's next box to then, if that is sooner,
  and owes it the spread. The age is the account id's (`Payer.Created`), so it counts from the
  account, not from the click, and a restart does not move it.
- **The first quiz comes the moment a young account's click empties its bank**, and a right
  answer is shields. `bonus_click` sits **outside** `throttle_click` so it reads the bank the
  click left: `Click.Dry` is `out.Limited && out.Budget.Empty()`, fewer than one click, which is
  when the client stops sending. The quiz is due at once and the next sweep (1s) asks it. While
  quizzes are off (`quiz.enabled`), none is owed.
- **Never a refill or a bomb.** `newAccountClicks` exists because one phone made an account a
  minute for a fresh bank; a welcome refill would hand that back, and a bomb would be a free strike
  per account. A spread click still costs a click, and a shield is placed with none, so a
  player with an empty bank can still use it. **A welcome kind that cannot be
  offered** (weight 0 in `bonuses.kinds`, or a full stack) **gives no early offer**: the turn goes
  back to the ordinary window, never to another kind.
- **Once per account**: `Registry.newcomers` remembers each young account until it is no longer
  young, so neither a second click nor a second empty bank pulls anything forward again. It is
  memory only, so a restart can welcome an account a second time while it is still young.
- Everything else is the ordinary offer: the stream must be open, the entrant active, the
  hourly caps not reached, and the guests behind a scope share one entrant, so a private tab a
  minute buys no more than the caps.
- **Off with `within` 0s.** An account welcomed keeps its welcome when the rule changes; a new rule applies
  to the accounts it meets after.
