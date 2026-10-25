# Shared protobuf contract

`proto/` is the **single source of truth** for the API contract — every app generates its own bindings from it (nothing here is hand-copied between apps). One package per bounded context:

- `proto/planet/v1/planet.proto` — the tile game (`ClickService`)
- `proto/chat/v1/chat.proto` — the live chat (`ChatService`)
- `proto/auth/v1/auth.proto` — who a caller is (`AuthService`): the mint that gates `Click`, the identity token resumed from the cookie, the caller's account, sign-in with Google, Discord or an emailed code, sign-out and deletion
- `proto/session/v1/session.proto` — the deprecated mint (`SessionService`), served by the same auth module until no client calls it
- `proto/player/v1/player.proto` — a player's name, color, stats, titles and the countries it plays for and against, and who is playing (`PlayerService`); the colors themselves are `NameColor` in `color.proto`, and a title is `Title` in `title.proto`, which `chat.proto` and `seasons.proto` import too
- `proto/seasons/v1/seasons.proto` — when the current season ends and its finale starts, who leads it, and the countries' daily rounds (`SeasonService`)
- `proto/support/v1/support.proto` — what a player sends the operator: a feature suggestion or a bug report, with photos or videos, or a report of another player (`SupportService`)
- `proto/clips/v1/clips.proto` — what the frontend's clip generator hands the backend's job that posts its clips: the stories worth a clip with the facts its model editor judges them by, and what to post with each one. No service: both are written as JSON files in one workflow run. Beside it, `admin.proto` is the numbers of the clips the job posted, a snapshot a day (`clips.v1.AdminService`): the job's `clips stats` keeps them over ssh, the backoffice reads them and makes the table

Beside them, `internal.proto` is what one backend module asks another, or `auth`'s admin gate asks `auth` (`auth.v1`, `player.v1`, `planet.v1`), and `events.proto` is what one tells the others in process (`planet.v1.TileTaken`, `auth.v1.AccountDeleted`). Neither is on the public router; the frontend generates both without using them. `admin.proto` is what an operator calls (`AdminService`, in `auth.v1`, `chat.v1`, `clips.v1`, `planet.v1`, `player.v1` and `support.v1`): over SSH on the console listener, and from the admin page on the admin listener, which only a signed-in operator passes.

Connect derives each service's route from its proto package, so a new context gets its own path with no prefix to allocate. Every `buf.gen.yaml` input points at the whole `proto` directory, so a new package is picked up by each generator with no config change.

After editing a `.proto`:

```bash
cd apps/backend && make proto   # regenerates apps/backend/generated/proto
cd apps/frontend && npm run proto   # regenerates apps/frontend/src/gen/grpc
cd apps/backoffice && npm run proto   # regenerates apps/backoffice/src/gen/grpc
```

Every app's `buf.gen.yaml` reference `../../proto` (or `../../../proto` for the backend, whose config lives one level deeper at `apps/backend/proto/`) — do not create per-app copies of the `.proto` files again.
