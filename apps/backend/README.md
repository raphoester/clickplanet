# ClickPlanet — Backend

The Go API of [clickplanet.lol](https://clickplanet.lol), a real-time multiplayer game where countries compete to own
tiles on a 3D globe. One process holds fifteen modules; it needs postgres and Redis.

The docs are in [`docs/`](docs), indexed by [`CLAUDE.md`](CLAUDE.md). Start with
[Commands](docs/commands.md), [Configuration](docs/configuration.md) and
[Fifteen bounded contexts, one process](docs/bounded-contexts.md).

```bash
make test       # Docker for the postgres and Redis suites
make deadcode
make lint
```

To run the whole stack on one machine, see [Local full stack](../../docs/local-stack.md).
