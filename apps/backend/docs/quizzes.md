# Quizzes (the quizzes module: `internal/quizzes/`)

A banner appears over the planet: press it and you get a question with three
choices and **a short clock** to answer. A right answer is worth the same charge a
caught box is, which the bonuses module pays. A wrong one, and running out of time, cost nothing. The module is
[The quizzes](quizzes-module.md); this page is the feature.

**It is a second way to earn a charge, not a box of another shape.** Its own
clock (the quizzes' rule set's `minInterval` / `maxInterval`; see [Its rule sets](quizzes-module.md#its-rule-sets)), its
own banner lifetime, its own `missRetry`, and its own `maxRightAnswersPerHour` beside the boxes' cap. The miss rule is
the box's: a banner that lapses, opened or not, brings the next one forward once, and an answer, right or wrong, clears
it. What it shares with a box is the reward: **the bonuses module draws it when it hears the right answer**, from
`kinds`, among the kinds the hand has room for, exactly as a box's ([A right answer](bonuses.md#a-right-answer)). Since
2026-11-13 a quiz is asked whatever the hand holds: the quizzes know no kind, and a right answer whose hand has room for
nothing is paid nothing.

**A box and a quiz no longer keep apart.** From 2026-11-02 to 2026-11-13 neither was offered while the other was out, nor
less than half the shorter of their shortest waits after it, since two clocks of 4-8 min drew a quiz within 30 s of a
box one time in six. It was the only rule that tied the two, and the owner dropped it when the quizzes left `bonuses`:
the page can hold one back if both land at once.

**The schedule and the bank live in the quizzes module, and only its registry draws from the bank**
(`TestTheBankIsTheRegistrysAlone`; `module.go` only loads it). A caller is the same thing as for the boxes — the streams
it has open, when it last clicked — and each module keeps its own copy of it, free to drift apart.

**A short window is the feature; the exact number is not.** The whole point is
to be answered from what somebody knows rather than from what they can look up,
so `quiz.answerWindow` is free to be retuned — the guideline is only that
it stay short enough that looking the answer up is not worth it. It is 8s today.
Everything below exists to make whatever it is set to real:

- **Two calls, not one.** `OpenQuiz` reads the question and is what **stamps the
  deadline**; `AnswerQuiz` compares against it. A banner can therefore sit
  unopened for its whole `offerTTL` without burning a second, and the seconds a
  player gets are their own.
- **The deadline is the server's stamp**, checked when the answer lands. A
  client that holds its own countdown open cannot spend longer than it was given.
- **Opening twice is the same question and the same deadline.** A reload is not a
  second window, and it is not a way to see a second question either.
- **The right answer never leaves the server until it is spent.** The registry
  holds the whole `bank.Round` and compares an index; `OpenQuiz` sends the
  text and the choices and nothing else. The bank itself is embedded in this
  binary and **deliberately never served to the browser** — see
  [`/quiz/README.md`](../../../quiz/README.md).
- **The banner says nothing about the question.** Not the text, not the choices,
  and not the subject either. It carried the subject country once, so the client
  could fly a flag on it — and that flag was the answer to **417 of the bank's
  1014 questions**: every `capitalOf` ("Tallinn is the capital of which
  country?", 191/191) and every `mostPeople` ("which of these has the most
  people?", 205/205). The fix is not safer templates. A teaser that has to be
  checked against every question in the bank leaks again the first time a
  template is added, so the banner carries a token and a deadline and nothing
  else. Planet's `QuizOffered` kept field 3 `reserved` for it; the offers' and the quizzes' (`quizzes.v1.QuizOffered`)
  never had one.
- **An answer cannot be sent without the question having been read.** A token
  answered before it was opened fails: that is a client guessing at three choices
  it was never sent.
- **The page ends a question the player leaves** (another window over it, such as a phone's assistant reading the
  screen, or the tab hidden) by sending its timeout at once: see the frontend's
  [Quizzes](../../frontend/docs/quizzes.md). The server cannot see the focus, so this is the page's rule alone, and
  what it sends is a plain timeout.
- **Each bank entry carries five wrong answers and a round shows two**, drawn when
  the quiz is offered (`Bank.Draw`). The same question is not the same three choices twice, so "press the
  second one for Estonia" is worth nothing.

**Every answer is counted for its question's rating** ([How hard each quiz question is](quiz-difficulty.md)): right,
wrong, the page's timeout, and an answer past the deadline, a caller's with no account too. A banner never opened and a
question opened and left are not. Nothing of it reaches the page: no difficulty is on the banner, the question or the
answer. **An answer that lands after its opened question lapsed, up to 10 s past the deadline,** is counted wrong, and still answered
`NotFound`, as before: the page's own timeout often reaches the server a trip after the deadline.

**An answer pays nothing in itself.** `AnswerQuiz` says right or wrong and which choice was right; a right one is told
(`QuizAnswered`) and the bonuses module pays it, a second or so later, on planet's stream (`charges_granted`).

**A wrong answer is not a failed call.** It resolves, grants nothing, and says
**which choice was right** — that is the only thing the feature gives back when
it gives nothing else, and the bank is generated public data, so there is nothing
worth keeping past the answer. Running out of time is sent by the client as a
choice past the end of the three: the server reads it as wrong, which it is, and
answers with the right one.

**The draw leans on the leaderboard.** The subject is the bank's, never the
client's — it steers the draw here and rides the broadcast after a win, and it is
on nothing a player sees before answering. A question's subject country is picked
with weight `1 + leaderBias × (its share ÷ an even share)`, the lean in force passed to `Bank.Draw`, read off
the same shares the toll prices a click from, as planet answers them — so there is no second
leaderboard to keep in step, and the board it reads is at most 10 s old rather than a
snapshot taken at boot. The floor of 1 keeps every country in the draw and the
lean is capped, because a quiz that only ever asked about the top three would be
four questions deep by the end of the week. **`leaderBias` 0 is a flat draw.** The bank reads the board through its
own port, `bank.Shares`, which `bank/rpc_planet_shares` answers: a runner asks planet's `GetTerritories` every
10 s and keeps each flag's share, so a draw under the registry's lock only reads what was kept. Every share is 0, a flat
draw, until planet first answers, and a failed ask keeps the last shares. Until 2026-11-11 planet's tile map answered it
in process.

**The lean only reaches a question that names its subject** (`namesSubject` in
the bank, worked out by the generator). "Sofia is the capital of which country?"
is about Bulgaria and answered by it: leaned, it made "press the leader" a way to
answer, and "Bulgaria" came up as the answer 23 times as often as on a flat draw.
So each subject's questions are two pools: the named ones weigh the lean, the
others weigh what they would on a flat draw. With no lean the draw is the same as
before.

**The draw can weigh each question by how hard it is**, inside the pool the lean picked, and never across subjects:
the quizzes' `width` and its values ([The draw](quiz-difficulty.md#the-draw)). Its `width` 0, default's, is off,
and the draw is the one above.

**`quiz_offered` is a case of the quizzes' own stream** (`quizzes.v1.QuizEvent`; `PlanetEvent`'s 8 until 2026-11-11, then
`BonusEvent`'s 4 until 2026-11-13, both `reserved` now), addressed to one caller, and carrying only the token and the
expiry. A win is announced to every stream with `quiz_won`: the winner's flag and the question's subject (the offers'
`bonus_taken` carried it until 2026-11-13). **The subject is on the broadcast and not on the offer**, which is the whole
distinction: after a win there is nothing left to give away.

**Both quiz procedures are session-gated** (the quizzes' `NewSessionInterceptor`). `AnswerQuiz` because a right answer
earns a charge, and `OpenQuiz` not only for symmetry: opening is what starts the `answerWindow`, and the caller it is
started for has to be the caller that answers.

**Switched on and off from the quizzes' rule set** (`enabled`). The bank is loaded at
every boot, so a switch needs no restart. Off, no quiz is asked and the boxes fly
exactly as they did; a quiz asked already can still be opened and answered until it
lapses. Switched on, each caller's next quiz is a whole wait away, so a wait long
past is not a quiz for everyone at once. **Production runs it on**, at the boxes'
own 4m-8m and `maxRightAnswersPerHour: 12`, so a player is asked about as many questions
as they are flown boxes.
