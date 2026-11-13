# The quizzes (`internal/quizzes/`)

**The quizzes module asks the quizzes and pays nothing.** It owns the quiz schedule, the bank, its draw and lean, each
question's rating and the answer log. It tells each answer (`quizzes.v1.QuizAnswered`); what a right answer pays is the
bonuses module's, which hears it ([The offers](bonuses.md)). The module left `bonuses` on 2026-11-13, the first of three
cuts that leave bonuses with the reward alone, so that a reward can later depend on the player's level (the owner's
Claude Doc "Boxes and quizzes leave bonuses"). See [Why it left](#why-it-left-bonuses).

It is downstream of planet and player: it hears planet's clicks and player's forgotten accounts, and asks planet the
board and the freeze. **It speaks planet's and player's language alone**, and nothing it speaks to hears it back:
`cmd/api/modules_test.go` pins it (`TestTheQuizzesSpeakPlanetsAndPlayersLanguageAlone`).

## What it holds

| package | what |
|---|---|
| `internal/quizzes` | the registry (`quiz.go`, `welcome.go`, `entrant.go`, `caller.go`), its rules (`rules.go`, `message.go`) and the use cases `open_quiz`, `answer_quiz` and `listen_for_events`; moved from `bonuses/internal/bonuses` |
| `internal/bank` | the bank and its draw, moved from `bonuses/internal/quizzes`; it embeds `generated/quiz/bank.json` ([The quiz bank](../../../docs/quiz-bank.md)), and answers each question's rating over its prior; `rpc_planet_shares` answers the lean |
| `internal/ratings` | how hard each question is ([How hard each quiz question is](quiz-difficulty.md)): the rating and its step, the copy in memory (`Book`), the answer log's store, and the use cases `record_answer` and `forget_account` |
| `internal/rules` | the list of its parts (`parts.All()`) and its fixtures (`rules.Seeded()`); the machinery is `cprules` ([Shared](shared.md)) |
| `internal/subscribers` | the clicks heard from planet, the forgotten accounts heard from player |
| `internal/quizzesv1controller` | the handlers, the session interceptor and reader, the freeze interceptor |

**Only the registry, `module.go` and the operator's list read the bank** (`TestTheBankIsTheRegistrysAlone`): it holds the
answers.

Its postgres schema is `quizzes`: `quizzes.rule_sets`, `quizzes.rule_set_choice`, `quizzes.quiz_answers`,
`quizzes.quiz_questions` and `quizzes.outbox`. Its config is the `quizzes:` block: `database`, `redis`, and
`offers.forgetAfter` (5 min) and `offers.sweepInterval` (1 s). Production gives it one postgres connection, which the
alerts module gave up, so `max_connections` stays 42.

## The wire (`proto/quizzes/v1/`)

- **`QuizService`**, public, routed by Caddy (`/quizzes.v1.QuizService/*`):
  - `OpenQuiz`, `AnswerQuiz`: as the offers' were. **`AnswerQuiz` answers right or wrong and the right choice, and pays
    nothing**: the reward comes on planet's stream ([Charges](charges.md#a-grant-on-the-stream)). `NotFound` for a quiz not
    offered to the caller, gone or answered; `FailedPrecondition` with `quizzes.v1.FrozenRefusal` while frozen. The click
    token is checked (`cpconnect.Attested`, counted in `quiz_session_checks`).
  - `ListenForEvents`: a stream of `QuizEvent`: `heartbeat`, `quiz_offered` (to the caller alone) and `quiz_won` (to every
    stream: the winner's flag and the question's subject). No `rules_changed`: the page reads none of the quizzes' rules.
    A stream's session reader refuses nothing: a signed-in account's stream attends as that account, any other as its
    address. **A quiz goes only to a caller with this stream open.**
- **Events** (`events.proto`): `QuizAnswered`, with `bonuses.v1.QuizAnswered`'s first three fields and numbers, so a payload
  the offers told reads as the new type, plus `token` (it names the answer) and `welcome` (a newcomer's first quiz).
- **`InternalService`**: `PlayRuleSet`, `GetRuleSetNames`, `GetRuleSet`, for the gamemaster and promo.
- **`AdminService`**: `GetRules`, `SaveRuleSet` and `ListQuizQuestions`, which the backoffice's Box & quiz rules page and
  Claude call (`quizzes_get_rules`, `quizzes_save_rule_set`, `quizzes_list_quiz_questions`;
  [Claude as a connector](connector.md)).

## What it asks of planet and player

| port | the answer | when |
|---|---|---|
| activity (`Registry.Clicked`) | `planet.v1.ClickTried`, group `quizzes-clicks` | every click planet tried |
| `bank.Shares` | `planet.v1.InternalService/GetTerritories` | every 10 s, by a runner |
| the map's freeze | `planet.v1.InternalService/GetFreeze` | each opening and answer |
| a forgotten account | `player.v1.PlayerForgotten`, group `bonuses-answers-forgotten` | each account deleted |

- **Activity.** The clicks taken, struck or unchanged, as the offers keep them ([What `Clicked`
  hears](offers-and-wallet.md#what-clicked-hears)). A click with no account, or heard more than `activeWithin` after it
  was tried, is no sign of a player. The group is new: at the deploy it reads `ClickTried`'s backlog, which it drops.
- **The lean**, as before ([Quizzes](quizzes.md)).
- **A forgotten account**: its id is blanked on every answer it gave (`forget_account_usecase`). The group kept the
  offers' name, so it goes on where they left the stream.

## Its rule sets

**The quizzes keep rule sets of their own**, named like planet's and played by name by the gamemaster beside planet's,
the antibot's and the offers' ([The gamemaster](gamemaster.md)). `default` holds every value, another set only what it
changes. `quizzes.v1.Rules`, each value named by its path (`minInterval is 0s, and must be above 0`):

- `enabled`, `minInterval`, `maxInterval`, `missRetry`, `offerTtl`, `answerWindow`, `leaderBias` — the quiz's clock, as
  [Quizzes](quizzes.md) tells it
- `maxRightAnswersPerHour` — no quiz for a caller that answered this many right in the last hour (the offers'
  `quiz.maxChargesPerHour`, the same number: each right answer pays one grant)
- `target`, `width`, `floor`, `floorAfter`, `explore` — the draw weighed by how hard each question is
  ([The draw](quiz-difficulty.md#the-draw))
- `activeWithin` — no quiz for a caller that has not clicked for this long
- `welcome.within` — a newcomer's first quiz ([A newcomer's first box and first quiz](welcome-bonus.md))
- `frozen` — no quiz is offered, and every opening and answer is refused

**Promo cites them as `quizzes.<path>`** ([Promo](promo.md)). Each set taken is logged at Warn (`quiz rules in force`),
each save too (`admin quiz rule set saved` or `refused`).

**Where the sets came from.** Migration `20261113161646_from_bonuses` copies each of the offers' sets, under the same
name: its `quiz` block, and the `activeWithin`, `welcome.within` and `frozen` a quiz read too (`quiz.maxChargesPerHour`
becomes `maxRightAnswersPerHour`). On a database with no bonuses schema it seeds production's quiz half of
2026-11-13. The offers' sets keep their `quiz` block for this release, read no more (`bonuses.v1.Rules.quiz` is
`reserved`), so the copy reads it whichever module migrates first; a later release strips it.

## The freeze

**An opening and an answer are refused before the quiz is spent** (`quizzesv1controller.NewFreezeInterceptor`), with
`FailedPrecondition` and a `FrozenRefusal`, while the quizzes play a frozen set or planet's `GetFreeze` answers frozen.
A slot that comes due while frozen is lost: the next wait is drawn, and a newcomer's welcome is lost with it.

## An answer, told

`answer_quiz_usecase` settles the answer in the registry, counts it for its question's rating
([The answer path](quiz-difficulty.md#the-answer-path)), and **records `QuizAnswered` in its outbox**, in a transaction of
its own, for a caller with an account, right or wrong. A right one is then told to every stream (`quiz_won`).

- **The outbox**, because a right answer told is a right answer paid: a `QuizAnswered` Redis refused would be a charge
  lost, where the offers paid it at once before. The relay sends every second, so the reward reaches the page about a
  second after the answer ([Telling other modules what happened](composite-layer.md#what-may-not-be-lost-the-outbox)).
- **A record that fails answers the call with an error**, as a failed payment did: the quiz is spent.
- **Heard by** the bonuses module (`bonuses-right-answers`, which pays a right one), [missions](missions.md)
  (`missions-answers`) and [progression](progression.md) (`player-stats-quizzes`). The last two kept their groups and read
  what the offers' stream still held first (`cpredis.SubscribeMoved` from `bonuses.v1.QuizAnswered`).

**The registry no longer asks planet's room.** The offers offered a quiz only when a hand had room for a kind it could
pay; the quizzes know no kind. A right answer whose hand has room for nothing is paid nothing, and the page says the
inventory is full.

**What the cut dropped: a box and a quiz no longer keep apart in time.** It was the only rule that tied the two; the
owner chose to drop it rather than keep a link between two modules. A welcome no longer waits for a box to be gone.

## Metrics

Every metric kept its name: `quiz_offers_total`, `quiz_lapsed_total`, `quiz_answer_seconds`, `quiz_answers_total`,
`quiz_answers_recorded_total{outcome}`. New: `quiz_session_checks`, the openings' and answers' session verdicts, which
`bonus_session_checks` no longer counts.

## Why it left bonuses

The player level needs a reward that depends on the level, so the reward must sit downstream of `progression`. But
progression hears the offers (`BonusClaimed`, `QuizAnswered`), so the offers could not hear progression without a cycle.
The offers held three languages that need different places: the box and the quiz (activity, upstream of missions and
progression) and the reward (downstream). The quiz shared nothing with the box but the rule that kept them apart. The
next cut moves the boxes out the same way; what is left in `bonuses` is the reward, which will hear progression's
`LevelReached`. Rejected: a gate in planet, levels in bonuses, and progression commanding the offers (`OpenKinds`,
`SetLevel`), events in disguise.

## The deploy

A hard switch ([A server restart reloads the page](../../../docs/reload-on-restart.md)): the page reads `QuizService`
and the reward from `charges_granted`. **The page first, then the server**, as always. After it:

- `XGROUP DESTROY events:bonuses.v1.QuizAnswered missions-answers` and `player-stats-quizzes`, once each read what it left
  there, then `DEL events:bonuses.v1.QuizAnswered`.
- The offers' `quiz` block in `bonuses.rule_sets` stays until a later release strips it.

## Tests

The registry's (the schedule, the welcome, the pace, the freeze, the late answers), the use cases', the handlers', the
stream's and its sink's, the bank's and the ratings' as before, the rules package's (and `cprules`' contract suites on its
migrations), and the seam test.
