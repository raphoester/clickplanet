# Quizzes (`internal/planet/internal/quizzes/`, scheduled in `bonuses/quiz.go`)

A banner appears over the planet: press it and you get a question with three
choices and **a short clock** to answer. A right answer is worth the same charge a
caught box is. A wrong one, and running out of time, cost nothing.

**It is a second way to earn a charge, not a box of another shape.** Its own
clock (`bonus.quiz.minInterval` / `maxInterval`), its own banner lifetime, its
own `bonus.quiz.missRetry`, and its own `bonus.quiz.maxChargesPerHour` on top of
the boxes'. The miss rule is the box's: a banner that lapses, opened or not,
brings the next one forward once, and an answer, right or wrong, clears it. A
caller can be holding an unopened banner and a flying box at the same time. What
it *does* share with a box is the reward: the kind is drawn at offer time from
`bonus.kinds`, filtered by `offerable` exactly as a box's is, so nobody is ever
asked a question for a bomb they already hold, and a caller with nothing to gain
is asked nothing.

**The schedule lives in `bonuses`, the content in `quizzes`.** A caller is one
thing — the streams it has open, when it last clicked, which accounts play
behind it — and that bookkeeping should not exist twice, so `bonuses/quiz.go`
hangs a second clock off the same `caller` and rides the same sweep. What is
genuinely separate is the bank, which is big, embedded, and knows nothing about
schedules or charges.

**A short window is the feature; the exact number is not.** The whole point is
to be answered from what somebody knows rather than from what they can look up,
so `bonus.quiz.answerWindow` is free to be retuned — the guideline is only that
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
  holds the whole `quizzes.Round` and compares an index; `OpenQuiz` sends the
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
  else. `QuizOffered` field 3 is `reserved` for it.
- **An answer cannot be sent without the question having been read.** A token
  answered before it was opened fails: that is a client guessing at three choices
  it was never sent.
- **Each bank entry carries five wrong answers and a round shows two**, drawn at
  the open. The same question is not the same three choices twice, so "press the
  second one for Estonia" is worth nothing.

**A wrong answer is not a failed call.** It resolves, grants nothing, and says
**which choice was right** — that is the only thing the feature gives back when
it gives nothing else, and the bank is generated public data, so there is nothing
worth keeping past the answer. Running out of time is sent by the client as a
choice past the end of the three: the server reads it as wrong, which it is, and
answers with the right one.

**The draw leans on the leaderboard.** The subject is the bank's, never the
client's — it steers the draw here and rides the broadcast after a win, and it is
on nothing a player sees before answering. A question's subject country is picked
with weight `1 + bonus.quiz.leaderBias × (its share ÷ an even share)`, read off
the *same* `Share` the toll prices a click from — so there is no second
leaderboard to keep in step, and the board it reads is the live one rather than a
snapshot taken at boot. The floor of 1 keeps every country in the draw and the
lean is capped, because a quiz that only ever asked about the top three would be
four questions deep by the end of the week. **`leaderBias` 0 is a flat draw, and
0 is what leaving the key out means.**

**The lean only reaches a question that names its subject** (`namesSubject` in
the bank, worked out by the generator). "Sofia is the capital of which country?"
is about Bulgaria and answered by it: leaned, it made "press the leader" a way to
answer, and "Bulgaria" came up as the answer 23 times as often as on a flat draw.
So each subject's questions are two pools: the named ones weigh the lean, the
others weigh what they would on a flat draw. With no lean the draw is the same as
before.

**`quiz_offered` is one more case on `PlanetEvent`**, addressed to one caller like
`bonus_offered`, and carrying only the token and the expiry. A win is announced
with the existing `bonus_taken`, which grew a `quiz_subject_country_id` rather
than getting an event of its own: it is the same news — somebody won a charge —
reached a second way. **The subject is on the broadcast and not on the offer**,
which is the whole distinction: after a win there is nothing left to give away.

**Both quiz procedures are session-gated**, beside `ClaimBonus` on
`NewSessionInterceptor`'s list. `AnswerQuiz` because it grants the same charge a
box does, and `OpenQuiz` not only for symmetry: opening is what starts the five
seconds, and the caller they are started for has to be the caller that answers.

**Off unless switched on.** With `bonus.quiz.enabled` false no bank is loaded,
`Quizzing` is never called, and the boxes fly exactly as they did. **Production
runs it on**, at the boxes' own 4m-8m and `maxChargesPerHour: 12`, so a player
is asked about as many questions as they are flown boxes —
`deploy/vps/backend.yaml` says why each number is what it is.
