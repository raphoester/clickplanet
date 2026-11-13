# Quizzes

A banner at the top of the screen: press it and you get a question with three
choices and **a short clock**. A right answer is worth a charge, the same as a
caught box; a wrong one, and running out of time, cost nothing.

`src/app/quiz/` is the whole of it — `quizStore.ts` holds the state and its
verbs, `useQuizMaster.ts` links it to the backend, `Quiz.tsx` draws, `Quiz.css`
styles — plus `domain/quiz.ts` for the shapes and the two pieces of arithmetic a
countdown needs. The backend half is `QuizMaster` in `backends/backend.ts`, which `QuizzesBackend` implements against
`quizzes.v1.QuizService`: the banner comes on the quizzes' stream (`quiz_offered`), and `OpenQuiz` and `AnswerQuiz` are
its calls (see [The quizzes stream](backends.md#the-quizzes-stream)).

**The state is a [store](stores.md), `useQuiz`.** `Quiz` reads the state and
calls `open` and `answer` itself. `PlayScreen` calls
`useQuizMaster(quizMaster, country, play)` and does not read the state. Only
the sign-in nudge it runs reads whether a quiz shows, so `PlayScreen` renders again
when a quiz comes and when it goes, not at each phase. The hook gives the store
its link (the backend, the flag played and the sound player) and subscribes to
the feed; a new flag or player changes the link and does not subscribe again.

**It never touches `globe.ts`.** A quiz is DOM at the top of the screen, not an
object in the scene, so `useQuizMaster` subscribes to the feed itself rather than
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
idle → offered → opening → asking → answering → answered → idle
```

Each phase is a different thing on screen *and* a different thing to a player:
`offered` is an invitation that costs nothing to ignore, `asking` is a clock
already running, `answering` is a pick on its way to the server, which takes no
second answer. Anything that goes wrong — a token the server will not honour, a
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
the kind only once a right answer is told, so the reel shows every kind and stops
on none. **The reward comes a moment after the answer, from planet's stream**: the
answer says right or wrong, the result keeps the reel rolling (`awaiting`), and the
first grant planet tells after it (`listenForRewards`, `charges_granted` with what it
`gained`) is the bonus won, which spins in. A grant that comes before the answer is
back is kept for it. With none in `REWARD_WAIT_MS` (2.5 s) the hand had room for
nothing, and the result says the inventory is full. **A mission paid at the same
moment can be taken for the reward**, a rare and harmless mix-up: both are charges
the player holds. The result is its quiz's token, so a reward that comes, or the
wait that ends, leaves it to go away at `RESULT_MS` as before.

It draws no countdown of its own either, because it is free to ignore, and it
goes away by itself.

**The countdown bar starts at what is actually left, not at full.** The
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

**Leaving the page ends the question** (`leaving` in `Quiz.tsx`, the store's `forfeit`). Players said in the chat
(2026-11-12) that they answered with their phone's assistant: on a Samsung, a long press on the home button draws
Gemini or Circle to Search over the page, which reads the question off the screen and says the answer, inside the 8 s.
Anything that does that has to take the page's place, so while a question is open, the window losing the focus (an
assistant or screen search over the page, another app, a click outside the browser) or the tab being hidden answers it
at once as out of time: the token is spent before the answer can be read off the other window. The focus moving inside
the page (the chat's field) does not count: an element's own blur never reaches the window's listener, and the
listener checks the target is not an element. The result says "You left the quiz", and **its timer waits for the
player to be back** (`whenBack`: the page's next focus or press), so it is still there to read. A pick made before the
focus left stands.

- **It is a client's rule, not the server's.** A page that is changed, or a second phone held up to the screen, goes
  round it. It stops the press-and-hold on the same phone, which is what the chat said they did. The server sees a
  forfeit as the page's timeout, and the rating counts it the same way.
- **An honest player loses a question to a call or a tab switch**, which costs nothing, as a timeout does.

**The result says which one was right whether or not that was the one pressed.**
A wrong answer costs nothing, so the only thing left to give back is the answer.
It draws the three choices again where they were, the right one green and a
wrong pick red, **as list items and not buttons**: no choice on the result can be
pressed, so none of them reacts to the pointer.

**The result goes away by itself after `RESULT_MS`, and its close button takes it
away sooner.** On a phone the card covers the top of the globe, and a player who
has read the answer wants to play again. `dismiss` acts only on `answered`, and
the timer that was to clear the result finds another state and does nothing.

**The quiz and the lines of news all want the band at the top**, and the news
gives it up. The quiz publishes its bottom edge (`QUIZ_BOTTOM`, `--quiz-bottom`,
through `useEdge`) while it shows, and every line of news (`.bomb-news`, so
`BombNews`, `FortifyNews`, `WelcomeNews`, `UnderAttack` and the mission progress,
and `BattleOver` and `ConnectionLost`) sits 8px under the lower of it and the
status. Four seconds of news nobody presses moves, a question somebody is
answering does not. The edge is measured, not a fixed drop: a question on two
lines with three answers is about 290px tall, and the 212px the news once dropped
put it over the answers (2026-11-13). The sign-in nudge reads `quizShown`, so it
never opens over a quiz.

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

`giveQuiz()` in the console puts one up at once against the fake backend, beside
`giveBomb()` and `giveBonus()`. The fake's bank is three questions — it is there
to develop the banner and the card against, not to be played.
