# How hard each quiz question is (`internal/bonuses/internal/ratings/`)

**Every quiz question carries a rating, and every answer to it moves the rating.** The bank ran from "stupidly niche,
only a history nerd knows" to "what's the capital of France", and nothing recorded which question was answered:
`QuizAnswered` says only who answered and whether it was right. Since step C of [Offers and the wallet](offers-and-wallet.md)
(2026-11-11) the offers log every answer and rate every question, and the draw can offer questions that fit once the
operator switches it on ([The draw](#the-draw)).

A typical question is answered about once a week: the bank holds 7,196 questions (830 generated, 6,366 hand-written)
and production sees about a thousand answers a day. **So a question's starting guess, its prior, matters more than the
update rule**, and the bank carries one per question ([Priors](#priors)).

## The model

- **One rating per question, `d`**, higher is harder, and later one per player, `s`, on the same scale. 0 is the
  average.
- **The chance of a right answer is `P = 1/3 + 2/3 · σ(s − d)`.** The 1/3 is a lucky guess among the three choices a
  round shows, so a question nobody knows sits near 33% right, and an average question (`d = 0`) is answered right two
  times in three by an average player.
- **Ratings, not a raw share right**: once the draw matches levels, a share right is biased by who was asked. With every
  player at 0 the two agree; the rating is what stays true after.
- **Players are not rated yet: every player is `s = 0`.** So `d` is how hard a question is for a random player we know
  nothing about. Player ratings come later (below), and the answer log keeps what they need.
- A rating is kept between the chances **0.34 and 0.99** for an average player (`ratings.MinChance`, `MaxChance`):
  `d` from about −4.18 to 4.60. A question at a third or at 1 would be infinitely hard or easy.

## An answer's step

**Each counted answer moves the chance the rating gave it a step `w` of the way to what happened** (1 for right, 0 for
wrong), and the rating is read back off the moved chance (`Rating.Answered`):

```
P  = 1/3 + 2/3 · σ(0 − d)
P' = P + w · (y − P)          clamped to [0.34, 0.99]
d' = −logit((3 · P' − 1) / 2)
w  = max(1 / (n + n0), floor)   n: the answers counted, this one included
```

- **It is the log-likelihood's gradient step, taken exactly.** The Fisher-scoring step on the log-likelihood of `P` is
  `Δd = −w · (y − P) / P′`, which moves the chance by `w · (y − P)` to first order. Taking that move exactly on the
  chance scale, rather than the linear step on `d`, keeps a question near either end from overshooting to the other on
  one surprise, where `P′` is tiny.
- **With players at 0 it is a running mean**: after `n` answers the chance is `(n0 · prior + right) / (n0 + n)` exactly
  (`TestTheChanceIsARunningMeanOfTheAnswersWithThePriorCountedTenTimes`), the prior counting as `n0` answers.
- **"Running mean, then exponential."** For a question `n0 = 10` (its prior is worth ten answers, so its first answer
  weighs 1/11) and the floor is 1/200: from the 190th answer on, each answer weighs 1/200, and the rating follows its
  last two hundred answers or so, a question whose players or wording changed included. For a player (later) `n0 = 5`
  and the floor 1/50.

**The simulation it is held to** (`ratings/rating_test.go`): questions of levels −3 to 3, from priors of 0.4, 0.6 and 0.9,
answered 3,000 times each by average players, end at their true chance, forty of each within 0.015 on average and each
within 0.12 (past the floor one question wanders about 0.025 around it); a question answered at chance level ends at
0.34 to 0.40; and over a bank of 300 questions with wrong priors (off by 0.2 on average), 30 answers take at least half
the error away (to 0.07, measured) and 400 take it under 0.035 (0.02, measured).

## What counts

| counts | how |
|---|---|
| an opened question answered right or wrong | `AnswerQuiz` in time |
| a timeout the page sends: a choice past the three | wrong, `timed_out` |
| an answer that lands past the deadline | wrong |
| an answer that lands after the sweep took its question, up to 10 s past the deadline | wrong, as a late answer (below) |

| does not count | why |
|---|---|
| a banner never opened | the player never saw the question |
| a question opened then left to lapse (the tab closed) | "didn't know" cannot be told from "left" |
| an answer more than 10 s past its deadline | a page that comes back that late was left, a phone locked or a tab asleep |
| an answer refused by the freeze or the session check | it never reached the offers |

- **A caller with no account counts for the question**, with no account recorded.
- **A late answer is counted, and the player is told what it was told before.** The page sends its timeout when its
  countdown ends, so it reaches the server about one trip after the deadline, and seconds later from a clock behind the
  server's. The sweep collects an opened question the second its deadline passes, so a timeout that lost that race was a
  lapse, and timeouts were lost to the log in proportion to the players' latency: the questions would have read easier
  than they are. **The registry keeps each opened question that lapsed until 10 s past its deadline** (`lapsedQuizzes`, `lateAnswers`), and
  an answer that lands then is counted wrong (`Answered.Late`) while `AnswerQuiz` answers `NotFound` as before. The
  lapse itself, its miss and the next quiz's wait are what they were.
- **The time to answer is from the opening**, which stamps the deadline (`pendingQuiz.openedAt`), not from the banner.

## The answer path

`answer_quiz_usecase` counts the answer right after the registry settles it, before it pays a right one, through
ratings' `record_answer_usecase` (`prom_record_answer` counts `quiz_answers_recorded_total{outcome}`,
`log_record_answer` logs a failure at Error):

1. The question's rating now: the one its answers moved it to, else its prior (`quizzes.Bank.Rating`).
2. **In one transaction (`InTx`)**: the answer's row in `quiz_answers` and the question's new rating in `quiz_questions`.
3. **After the commit, the copy in memory moves** (`ratings.Book.Keep`): an answer that did not commit moves nothing.

- **A question's answers are counted one at a time** (64 stripes by question), so two answers at once never both move
  it from the same rating, and a slow commit holds up only the questions of its stripe.
- **The count runs on a context the caller's leaving does not cut**, 5 s at most: the answer is spent.
- **A count that fails costs the player nothing**: the answer stands, the charge is paid, and the question is one answer
  short, logged and counted as `failed`.
- **Unchanged**: `QuizAnswered`, the page, the answers, every metric before. No difficulty reaches the client.

## The tables (schema `bonuses`)

- **`quiz_answers`**, one row per counted answer, never edited: `question_id`, `account_id` (null for a caller with
  none, or once forgotten), `correct`, `timed_out`, `answer_ms` (from the opening), `answered_at`. **It is the source of
  truth**: kept so the ratings can be recomputed with a better model, the players' among them. Never pruned: a thousand
  rows a day.
- **`quiz_questions`**, one row per question answered at least once: `question_id`, `rating`, `answers`,
  `right_answers`. The log's running totals.
- **A forgotten account** (`player.v1.PlayerForgotten`, group `bonuses-answers-forgotten`) has its id blanked in
  `quiz_answers` (`forget_account_usecase`). Its answers stay, and so do the ratings they moved.

## The boot

**The draw never touches postgres.** At boot the module reads every row of `quiz_questions` once into a
`ratings.Book`, and the bank answers each question's rating from it, over the question's prior: a question with no row
is at its prior. Only the answer path writes the book, once each answer committed. `quiz bank loaded` logs how many
questions the book holds (`rated`).

## Priors

**`quiz/bank.json` may carry `"prior"` on a question: the share of players expected to get it right**, between a third
and 1. The bank's `format` stays 1: the field is optional, and a field the build does not know is read past, so a bank
regenerated with priors loads on this build and an older one.

- **A question with no prior starts at 0.6** (`quizzes.FallbackPrior`): more often right than wrong, as most of the
  bank's questions are written to be.
- **Its starting rating is `d0 = −logit((3 · prior − 1) / 2)`**, the prior kept between 0.34 and 0.99, so a prior of a
  third or of 1 stays finite. A prior under a third (a trick question) reads as 0.34.
- **A prior that is not a share** (under 0 or over 1) refuses the boot, as any broken question does.
- **A prior changed later moves only the questions with no row yet.** A question already answered keeps the rating its
  answers moved, which started from the old prior. Correcting the priors as a whole is a replay of the log (below).
- The parallel task that rates the bank writes the field from the generator (`apps/frontend/scripts/quiz/`); see
  [The quiz bank](../../../docs/quiz-bank.md).

## The draw

**The draw weighs a question by how hard it is, inside the pool the lean picked, and ships switched off.** Five values
of the offers' rule sets (`bonuses.v1.Rules.quiz`, played per set like every offer rule: see
[Its rule sets](bonuses.md#its-rule-sets)), read as `quizzes.Fit`:

- `width`: how wide the bell around the target is, on the rating scale. **0 is off, and is `default`'s value**: the
  draw of before, with no floor and no exploring. A question a width from the target weighs 0.61 of one on it, two
  widths away 0.14. 1 is a fair start: with a target of 0.7 (a rating of −0.2), a question answered right 54% or 85%
  of the time (0.8 and −1.2) weighs 0.61, and one at 43% or 93% (1.8 and −2.2) weighs 0.14.
- `target`: the chance of a right answer, for an average player, the draw aims at (0.7 in default), 0.34 to 0.99.
- `floor` and `floorAfter`: **a question its answers measured under `floor` (0.42), once at least `floorAfter` (15)
  moved its rating, is never drawn.** Measured is its rating, not its raw share right, which a later draw that matches
  levels would bias. A prior alone never leaves a question out, even one under the floor: it only weighs less, so a
  wrong prior is corrected by its answers. `floor` is at most `target`, and `floorAfter` above 0.
- `explore`: the share of draws (0.1) that ignore how hard a question is, so every question keeps being asked and
  measured. An exploring draw never asks a question under the floor either.

**How it draws** (`Bank.Draw(lean, fit)`):

1. **The subject-first draw and the leaderboard lean are exactly as before** ([Quizzes](quizzes.md)): each subject's
   named and unnamed questions are two pools, weighed by the lean and their sizes, and one pool is drawn.
2. Inside it, the questions under the floor are left out. One draw in `explore` takes one of the rest flat; the others
   weigh each by the bell, `exp(−((d − d_target) / width)² / 2)`, `d_target` being the rating `target` stands for.
3. **A pool with nothing left to draw** (every question in it under the floor) **is left out and the lean draws again
   among the rest**, so its subject's other pool and the other subjects keep their weights to each other. A bank with
   nothing to draw anywhere is drawn flat, floor ignored: a quiz is still asked.

`fit_test.go` holds it to the bell's shares, the floor and its exceptions, the exploring draws, the subject drawn as
before, and both cases of nothing to draw. **The operator switches it on from the backoffice** (Box & quiz rules, Quiz
difficulty) or Claude's `bonuses_save_rule_set`, after a few weeks of answers; no set does yet.

## The operator's list

**`bonuses.v1.AdminService/ListQuizQuestions`** answers every question of the bank with how hard it is, and the connector
makes it Claude's `bonuses_list_quiz_questions` with no code of its own ([Claude as a connector](connector.md)). No
backoffice page reads it yet.

- **Each question**: its id, text, right answer and subject; its prior (unset when the bank holds none: it started at
  0.6); its rating; the share of average players the rating predicts get it right (`predicted_share`); how many
  answers moved it and how many were right; and the share they measured (`measured_share`, unset with no answer).
- **Sorted** by the predicted share (`order` unset or `QUIZ_QUESTION_ORDER_PREDICTED`) or the measured one
  (`QUIZ_QUESTION_ORDER_MEASURED`, a question never answered last either way), the hardest first unless
  `easiest_first`, then by id. **Filtered** by `min_answers`: 15 lists the questions the floor could judge.
- **A page at a time**: `limit` (50 unless asked, 500 at most) from `offset`, and `total`, how many the filter keeps.
- **It reads memory, never postgres** (`quiz_questions_query.MemoryQuery`, [Reads are queries](queries.md)): the bank
  and the book the draw reads, through `inmemory_quiz_questions`, which the bank's seam test lets read the bank, answers
  and all (`TestTheQuizzesAreTheOffersAlone`). The list is the admin listener's alone.

## What comes later

- **Correcting the priors as a whole from the log**, after a few weeks of exploration: fit how far each kind of
  question's priors were off (by template, by subject, by the generator's own score), correct the bank's priors, and
  replay `quiz_answers` from the corrected priors into `quiz_questions`.
- **Player ratings, seeded by replay**: once the questions' ratings settle, replay the log in order with the players'
  step (`n0 = 5`, floor 1/50), each answer moving its player against the question's settled rating, then keep both
  moving live. The log has what that needs: the account, the question, right or wrong, the timeout, and the order.
- **Matching only part of the way toward a player's level**, so that an expert still wins more than a beginner: aim the
  draw at a level between the average's and the player's.
