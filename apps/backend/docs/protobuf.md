# Protobuf

API contracts live in the monorepo-shared [`/proto`](../../../proto) (also used by the frontend and the backoffice), one package per bounded context (see [Shared protobuf contract](../../../docs/proto.md)). A module's public service sits in its own `.proto`, its events in `events.proto`, what it is asked to change in `commands.proto`, what other modules call in `internal.proto`, and the operator's calls in `admin.proto`. Generated code goes to `generated/proto/`. Use `make proto` to regenerate after editing `.proto` files (requires the `buf` CLI, plus `protoc-gen-go` and `protoc-gen-connect-go` on `PATH`).

`generated/` is for everything the root owns and this app carries a committed copy of, because the Docker build context is this directory: `generated/proto` from [`/proto`](../../../proto) via `make proto`, `generated/descriptors` (every `admin.proto` with its comments, for the [connector](connector.md)) by the same target, `generated/map` from [`/map`](../../../map) via `make map` — see [Map geography](map-geography.md) — `generated/quiz`
from [`/quiz`](../../../quiz) via `make quiz` (see [Quiz bank](../../../docs/quiz-bank.md)), and `generated/moderation`,
the policy [`/docs/moderation.md`](../../../docs/moderation.md) the chat's judge sends Claude, via `make moderation`. Nothing in there is edited by hand; run the target.

The proto package is the **only** version number: Connect derives each route from it, and the controller and connect package names follow — `planet.v1` gives `planetv1controller` and `planetv1connect`, `chat.v1` gives `chatv1controller` and `chatv1connect`. There is no gRPC here — Connect serves the service definitions over ordinary HTTP/1.1 POSTs (and h2c, for clients that want it).
