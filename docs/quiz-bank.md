# Shared quiz bank

`quiz/` is one of the four things the root holds for the apps, beside `proto/`, `map/` and
[`docs/moderation.md`](moderation.md), and, like the last, it is shared with only *one* app. `bank.json` is
every question the quizzes can ask and the answer to each; `make quiz`
copies it into `apps/backend/generated/quiz/`, where it is embedded, and the backend's `bonuses` module alone reads it
(`internal/bonuses/internal/quizzes`; planet's `internal/quizzes` until 2026-11-11).

**The frontend deliberately has no copy**, because the answers are in it: a bank served to the page
is a bank anyone can fetch with the network tab open. The question and its three choices reach the
client one at a time from `bonuses.v1.BonusService/OpenQuiz`, and which of the three is right never leaves the server.

```bash
cd apps/frontend && npm run quiz:generate   # rewrite /quiz/bank.json
cd apps/backend && make quiz                # → apps/backend/generated/quiz (embedded)
```

**A question may carry `prior`**, the share of players expected to get it right, from a third to 1: the backend starts
the question's rating there, and a question with none starts at 0.6. The field is optional, so the bank's `format`
stays 1 and a backend reads a bank with or without it (see [How hard each quiz question is](../apps/backend/docs/quiz-difficulty.md#priors)).

**Where the priors come from: `quiz/priors.json`.** One entry per question: its id, the text and answer it was rated
with, and the prior. The generator sets `prior` on a question only when its id, text and answer all match the entry, so
a question whose wording changed starts at 0.6 until it is rated again. The file is one question per line, like the bank.

The priors in the file were rated by hand: subagents each read a tenth of the bank and judged each question for a
teenager who has not studied its subject, against two anchors ("What is the capital of France?" about 0.95, a question
about an obscure Pacific island prince about 0.38). Then one shift on the rating scale (`d = -logit((3P - 1) / 2)`) moved
the whole bank so its mean prior is 0.45, and every prior is clamped to 0.34 to 0.99. The shift is recorded in the file.

To rate new questions, the same way: read them, write a prior per id into `quiz/priors.json`, then run the generator
and `make quiz`:

```bash
cd apps/frontend && npm run quiz:generate   # the bank, with the questions that changed
# add the new or changed ids to quiz/priors.json
cd apps/frontend && npm run quiz:generate   # write the priors into the bank
cd apps/backend && make quiz                # → apps/backend/generated/quiz, then commit bank, priors and the copy
```

About 820 of the questions are **derived** from the same pinned Natural Earth snapshot `map/` is cut
from and from the tile borders themselves, so they are regenerated rather than corrected; the rest
are hand-written in `apps/frontend/scripts/quiz/extra.json` for what no dataset here can answer. See
[`quiz/README.md`](../quiz/README.md), which also has the rule that keeps the hand-written half from
rotting.
