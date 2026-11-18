# The map tools

`src/map/`, one page: **Download a replay** (`GetReplay`). A span back from an end (empty: now), saved as
`replay-<since>-<until>.json`, the answer as protobuf JSON, which the frontend's `npm run clip` reads as it reads what
`clips fetch` saves. The ledger keeps 72 hours. The file is made in memory (`download.ts`, a `blob:` link), which the
CSP allows.

**The page draws nothing of it, and the file keeps its eras** (`GetReplayResponse.eras`): the board the world played at
the span's start, then each it switched to in the span, with when. An event's tile ids are the board of the era in
force at its time, and the file is the whole answer, so whatever plays it can place each event on its board. The line
the page says once it is saved names the boards (`erasOf`, `boardsOf`): "on earth-1", or across a switch each era with
its start and its events, "on earth-1 (2 events), then on earth-2 from <the switch, in the operator's time> (3 events)".
