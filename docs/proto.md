# Shared protobuf contract

`proto/` is the **single source of truth** for the API contract — every app generates its own bindings from it (nothing here is hand-copied between apps). One package per bounded context:

- `proto/planet/v1/planet.proto` — the tile game (`ClickService`)
- `proto/chat/v1/chat.proto` — the live chat (`ChatService`)
- `proto/auth/v1/auth.proto` — who a caller is (`AuthService`): the mint that gates `Click`, the identity token resumed from the cookie, the caller's account, sign-in with Google, Discord or an emailed code, sign-out and deletion
- `proto/session/v1/session.proto` — the deprecated mint (`SessionService`), served by the same auth module until no client calls it
- `proto/player/v1/player.proto` — a player's name, color, stats, titles and the countries it plays for and against, and who is playing (`PlayerService`); the colors themselves are `NameColor` in `color.proto`, and a title is `Title` in `title.proto`, which `chat.proto` and `seasons.proto` import too
- `proto/seasons/v1/seasons.proto` — when the current season and its last day end, who leads it, and the countries' rounds, days and rounds of their own, with a battle's numbers (`SeasonService`). Beside it, `rules.proto` is `seasons.v1.Rules`, how seasons counts points while a set plays (a round of its own and its times), which postgres keeps as protojson, and `admin.proto` the operator's calls on it (`seasons.v1.AdminService`: `GetRules`, `SaveRuleSet`)
- `proto/missions/v1/missions.proto` — the day's three missions of a flag, how far the caller took them, and which are paid (`MissionService`); each kind of mission is its own message in a `oneof`, so a new kind adds a message and no field
- `proto/support/v1/support.proto` — what a player sends the operator: a feature suggestion or a bug report, with photos or videos, or a report of another player (`SupportService`)
- `proto/metagame/v1/metagame.proto` — what the schedule plays now, in the players' words: the `Turn`, the kind of the running entry (`EntryKind`: plain, or battle), the operator's description of its rules, since when and until when, and the next battle to come, never a rule set's name (`MetagameService`: `ListenForEvents`, the turn then each change of it, and `GetTurn`, the same as a cached GET). Beside it, `admin.proto` is the rule schedule: which rule set planet, the antibot and seasons play, of which kind, with which words for the players, and when (`metagame.v1.AdminService`: `GetSchedule`, `AddEntry`, `EditEntry`, `DeleteEntry`)
- `proto/antibot/v1/admin.proto` — the operator's bans and rules (`antibot.v1.AdminService`: `BanPlayer`, `UnbanPlayer`, `InspectPlayer`, `GetRules`, `SaveRuleSet`). No public service: the antibot hears planet and answers only the operator. Beside it, `rules.proto` is `antibot.v1.Rules`, every bound of a rule set, which postgres also keeps as protojson
- `proto/clips/v1/clips.proto` — what the frontend's clip generator hands the backend's job that posts its clips: the findings its model editor makes the stories from (every front, tour and place measured, and the season's stories), the story the job hands back to render (`Pick`), and what to post with each clip. No service: all are written as JSON files in one workflow run. Beside it, `admin.proto` is the numbers of the clips the job posted, a snapshot a day (`clips.v1.AdminService`): the job's `clips stats` keeps them over ssh, the backoffice reads them and makes the table

Beside them, `internal.proto` is what one backend module asks another, or `auth`'s admin gate asks `auth` (`auth.v1`, `player.v1`, `planet.v1`, `antibot.v1`, `seasons.v1`), `events.proto` is what one tells the others over Redis Streams (`planet.v1.TileTaken`, `auth.v1.AccountDeleted`), and `commands.proto` is what one asks another to change, in the receiver's words (`planet.v1.GrantCharges`, which missions sends, `planet.v1.RefuseCaller`, which the antibot sends, and `planet.v1.PlayRuleSet`, `antibot.v1.PlayRuleSet` and `seasons.v1.PlayRuleSet`, which the metagame module sends). None is on the public router; the frontend generates them without using them. `admin.proto` is what an operator calls (`AdminService`, in `antibot.v1`, `auth.v1`, `chat.v1`, `clips.v1`, `planet.v1`, `player.v1`, `metagame.v1`, `seasons.v1` and `support.v1`): over SSH on the console listener, and from the admin page on the admin listener, which only a signed-in operator passes.

Connect derives each service's route from its proto package, so a new context gets its own path with no prefix to allocate. Every `buf.gen.yaml` input points at the whole `proto` directory, so a new package is picked up by each generator with no config change.

After editing a `.proto`:

```bash
cd apps/backend && make proto   # regenerates apps/backend/generated/proto
cd apps/frontend && npm run proto   # regenerates apps/frontend/src/gen/grpc
cd apps/backoffice && npm run proto   # regenerates apps/backoffice/src/gen/grpc
```

Every app's `buf.gen.yaml` reference `../../proto` (or `../../../proto` for the backend, whose config lives one level deeper at `apps/backend/proto/`) — do not create per-app copies of the `.proto` files again.
