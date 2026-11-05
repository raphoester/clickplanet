# Shared quiz bank

`quiz/` is the **third and last thing the root holds**, and the one that is shared with only *one*
app. `bank.json` is every question the quizzes can ask and the answer to each; `make quiz`
copies it into `apps/backend/generated/quiz/`, where it is embedded.

**The frontend deliberately has no copy**, because the answers are in it: a bank served to the page
is a bank anyone can fetch with the network tab open. The question and its three choices reach the
client one at a time from `OpenQuiz`, and which of the three is right never leaves the server.

```bash
cd apps/frontend && npm run quiz:generate   # rewrite /quiz/bank.json
cd apps/backend && make quiz                # → apps/backend/generated/quiz (embedded)
```

About 820 of the questions are **derived** from the same pinned Natural Earth snapshot `map/` is cut
from and from the tile borders themselves, so they are regenerated rather than corrected; the rest
are hand-written in `apps/frontend/scripts/quiz/extra.json` for what no dataset here can answer. See
[`quiz/README.md`](../quiz/README.md), which also has the rule that keeps the hand-written half from
rotting.
