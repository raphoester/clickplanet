# A newcomer's first box and first quiz (`boxes/welcome.go`, `quizzes/welcome.go`)

**An account made less than `welcome.within` ago (10m) gets its first box and its first quiz early, once each.** The box
is the boxes' (`boxes.v1.Rules.welcome`, copied from the bonuses module's on 2026-11-15), the quiz the quizzes'
(`quizzes.v1.Rules.welcome`, copied on 2026-11-13), each with its own `within`. Measured from 2026-10-02 to 10-22: 36% of new accounts left
inside two minutes, most at the 10th or 11th click (`clicks.newAccountClicks`), and
the first box and quiz came 4-8 minutes in, so they never saw one.

- **The first box is a spread, `welcome.boxAfter` (1m) after the account was made.**
  The first click of a young account pulls its entrant's next box to then, if that is sooner,
  and owes it an early box. The age is the account id's (`Click.Born`, which the boxes' `click_tried_subscriber` reads
  off the UUIDv7 id `ClickTried` names, `cpsession.AccountID.CreatedAt`), so it counts from the account, not from the
  click, and a restart does not move it. **The boxes tell the catch as a welcome** (`BoxClaimed.welcome`), and the bonuses
  module pays it a spread, or nothing ([A box caught](bonuses.md#a-box-caught)).
- **The first quiz comes the moment a young account's click empties its bank**, and a right
  answer is shields. Planet's `try_click` tells the bank the click left as `ClickTried.dry`
  (`out.Limited && out.Budget.Empty()`, fewer than one click, which is when the client stops sending), and the quizzes
  read `Click.Dry` off it. The quiz is due when that click is heard, about 0.1 s after it was tried, and the next sweep
  (1s) asks it. A `ClickTried` Redis refused is a dry click never heard: the early quiz waits for the next. While
  quizzes are off (`enabled`), none is owed. **The quizzes tell the answer as a welcome** (`QuizAnswered.welcome`), and the
  bonuses module pays a right one shields, or nothing when the stack of shields is full ([A right
  answer](bonuses.md#a-right-answer)): since 2026-11-13 the quiz is asked even then, since the quizzes know no kind.
- **Never a refill or a bomb.** `newAccountClicks` exists because one phone made an account a
  minute for a fresh bank; a welcome refill would hand that back, and a bomb would be a free strike
  per account. A spread click still costs a click, and a shield is placed with none, so a
  player with an empty bank can still use it. A guest gets its one refill when it signs in
  ([A guest's first sign-in](sign-in-refill.md)): that needs a new identity, not a new account. **A welcome box whose spread
  cannot be paid** (weight 0 in `kinds`, or a full pool) **pays nothing**, never another kind. Until 2026-11-15 it gave no
  early box: the turn went back to the ordinary window. The boxes know no kind, so they give no early box only to a hand
  with room for no kind at all. A welcome quiz is asked all the same and its right answer paid nothing, never another
  kind.
- **A full hand is the wallet's to say** (`GetRoom`, [Offers and the wallet](offers-and-wallet.md#room)). An account
  whose first click lands while the room is asked waits for the next sweep, so its welcome is not lost to that race.
- **Once per account**: each registry's `newcomers` remembers each young account it welcomed until it is no longer
  young, so neither a second click nor a second empty bank pulls anything forward again. It is
  memory only, so a restart can welcome an account a second time while it is still young.
- **A welcome waits for nothing**: since 2026-11-13 a box and a quiz no longer keep apart, so the welcome box and the
  welcome quiz come at their own moments, together if they fall together.
- Everything else is the ordinary offer: the stream must be open, the entrant active, the
  hourly caps not reached, and the guests behind a scope share one entrant, so a private tab a
  minute buys no more than the caps.
- **Off with `within` 0s.** An account welcomed keeps its welcome when the rule changes; a new rule applies
  to the accounts it meets after.
