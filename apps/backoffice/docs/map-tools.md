# The map tools

`src/map/`, one page: **Download a replay** (`GetReplay`). A span back from an end (empty: now), saved as
`replay-<since>-<until>.json`, the answer as protobuf JSON, which the frontend's `npm run clip` reads as it reads what
`clips fetch` saves. The ledger keeps 72 hours. The file is made in memory (`download.ts`, a `blob:` link), which the
CSP allows.
