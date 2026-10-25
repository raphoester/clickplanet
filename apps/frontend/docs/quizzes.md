# Quizzes

A banner at the top of the screen: press it and you get a question with three
choices and **a short clock**. A right answer is worth a charge, the same as a
caught box; a wrong one, and running out of time, cost nothing.

`src/app/quiz/` is the whole of it — `useQuiz.ts` drives, `Quiz.tsx` draws,
`Quiz.css` styles — plus `domain/quiz.ts` for the shapes and the two pieces of
arithmetic a countdown needs. The backend half is `QuizMaster` in
`backends/backend.ts`.

**It never touches `globe.ts`.** A quiz is DOM at the top of the screen, not an
object in the scene, so `useQuiz` subscribes to the feed itself rather than
being handed offers down through the globe the way a flying box is. What a right
answer wins reaches the inventory the way every other charge does: the backend
holds the charges and tells whoever is listening.

**The client never knows an answer before it gives one.** The bank lives on the
server and is deliberately not shipped to the browser (see
[`/quiz/README.md`](../../../quiz/README.md)). `listenForQuizzes` brings a banner
carrying **a token and a deadline and nothing else**; `openQuiz` brings the
question and its three choices and **starts the server's clock**; `answerQuiz` is
the first thing that says which of the three was right.

**One state machine, `QuizState`:**

```
idle → offered → opening → asking → answered → idle
```

Each phase is a different thing on screen *and* a different thing to a player:
`offered` is an invitation that costs nothing to ignore, `asking` is a clock
already running. Anything that goes wrong — a token the server will not honour, a
stream that dropped — falls back to `idle`, because a quiz nobody can answer
should leave nothing behind. **The token rides through the state** rather than
sitting beside it, so there is no way to answer one quiz with another's token
while a banner and a question are changing places.

**A banner arrives only into an empty screen.** The server will not offer a
second, but a stale one arriving mid-question would take the clock away from
under somebody already reaching for a choice.

**The banner gives nothing away** — not the question, not the choices, and not
what it is about. It named the subject country and flew its flag once, which
read as a harmless teaser and was not: that flag was the answer to **417 of the
bank's 1014 questions**, every "Tallinn is the capital of which country?" and
every "which of these has the most people?". The fix is not to pick safer
templates, because a teaser that has to be checked against every question in the
bank leaks again the first time a template is added. What makes the banner worth
pressing is the charge behind it.

**The charge is teased as a slot reel** (`PrizeReel`): the five bonus boxes roll
past under a gold "?", on the banner and again beside the clock. The server picks
the kind when it offers the quiz and does not send it, so the reel shows every
kind and stops on none. A right answer is the first time the client knows, and
the box then spins in as the bonus won.

It draws no countdown of its own either, because it is free to ignore, and it
goes away by itself.

**The countdown bar starts at what is actually left, not at full.** The five
seconds are the server's and they began when it answered, so a slow round trip
has already spent some of them; a bar that started full would promise time the
player does not have. From there it is **one CSS animation on `transform`** to
empty, started that far in with a negative `animation-delay` — on the compositor,
so a whole window of continuous animation costs nothing beside a WebGL globe
drawing at the same time, where a `width` transition would relayout every frame.
An animation rather than a transition because **a pressed choice pauses it**
(`animation-play-state`): the bar stops where the answer was given. The seconds
beside it are drawn by React once a second (`secondsLeft`, `untilNextSecond`), and
in the last three the coin and the card's glow turn red. Under
`prefers-reduced-motion` the countdown **stays**: it is information, not
decoration.

**Running out of time is sent as a choice past the end of the three.** The server
reads it as wrong, which it is, and answers with the right one — so a question
nobody managed to answer still says what it was. There is no other way to learn
it.

**The result says which one was right whether or not that was the one pressed.**
A wrong answer costs nothing, so the only thing left to give back is the answer.
It draws the three choices again where they were, the right one green and a
wrong pick red, **as list items and not buttons**: nothing on the result can be
pressed, so nothing on it reacts to the pointer.

**The quiz and `BombNews` both want the band at the top**, and the bomb line is
the one that gives it up (`lowered`): four seconds of news nobody presses moves,
a question somebody is answering does not.

**Three sounds, one switch.** `quiz` when the banner arrives (the same reason
`bonusSpawn` exists: it is at the top of the screen and the player is looking at
the globe), `quizRight` and `quizWrong` when the answer lands. All three answer
to one `quiz` switch through `switchOf` — a quiz is one feature making three
noises inside ten seconds, and three lines in the settings panel for that is two
lines too many. `quiz` is the banner's sound, which is also what the panel plays
when the switch is turned on.

`quizWrong` is gentle on purpose, and rounder and higher than `refused`: a wrong
answer costs nothing, so it says "ah well" rather than "no". A sound that
punished a guess would make guessing feel expensive when it is free.

`useQuiz` takes the player and holds it in a ref, so a settings toggle does not
resubscribe the feed; it is optional, and a page with no sound wired still works.

`giveQuiz()` in the console puts one up at once against the fake backend, beside
`giveBomb()` and `giveBonus()`. The fake's bank is three questions — it is there
to develop the banner and the card against, not to be played.
