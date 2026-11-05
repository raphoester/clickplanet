# Protocol Buffers

Types are defined in the monorepo-shared [`/proto`](../../../proto), one package per
bounded context, and generated to `src/gen/grpc/<package>/v1/` — `*_pb.ts` for
the messages and `*_connect.ts` for the service client. `buf.gen.yaml` points at
the whole `proto` directory, so a new package needs no config change; run
`npm run proto` after changing a `.proto`.

- [`planet/v1/planet.proto`](../../../proto/planet/v1/planet.proto) — `ClickRequest`,
  `ClickBudget`, `GetMapResponse`, `TileUpdate`
- [`chat/v1/chat.proto`](../../../proto/chat/v1/chat.proto) — `ChatMessage`,
  `SendMessageRequest`, `GetHistoryResponse`
- [`auth/v1/auth.proto`](../../../proto/auth/v1/auth.proto) — the mint, the
  account and sign-in (`AuthService`)
- [`session/v1/session.proto`](../../../proto/session/v1/session.proto) — the
  deprecated mint, no longer called
- [`player/v1/player.proto`](../../../proto/player/v1/player.proto) — the
  username and who is playing (`PlayerService`)
- [`scoring/v1/scoring.proto`](../../../proto/scoring/v1/scoring.proto) — the
  books and rounds of points, read by the ids the gamemaster names (`ScoringService`)
- [`support/v1/support.proto`](../../../proto/support/v1/support.proto) — what a
  player sends the operator (`SupportService`)
- [`gamemaster/v1/gamemaster.proto`](../../../proto/gamemaster/v1/gamemaster.proto) — the
  kind of the schedule's entry that plays, a battle or a plain one
  (`GamemasterService`, see [Battles](battle.md))

`ChatMessage.sentAtUnixMs` is an `int64`, which `protoc-gen-es` gives you as a
`bigint` — `chatBackend.ts` converts it at the edge so nothing above it deals in
two number types.
