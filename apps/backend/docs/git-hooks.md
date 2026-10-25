# Git hooks

`./.githooks/install` points git at [`.githooks/`](../../../.githooks). It is a
script rather than a root `Makefile` target because this repo keeps build tooling
inside each app, and hooks are the one genuinely repo-wide thing.

- **pre-commit** — `make tidy` (re-staging only what was already staged, plus
  `go.mod`/`go.sum` if tidy moved them) and `make lint-ci`, plus the frontend's
  eslint, each only when that app has staged changes.
- **commit-msg** — conventional commits, which the history already uses, and a
  72-character subject so `git log --oneline` stays readable. Merge, revert,
  fixup and squash subjects are git's to format and are left alone.
- **pre-push** — `make test`, `make deadcode`, `make lint-ci` and
  `make check-format`, run concurrently; output is only printed for a step that
  fails.

All three take `--no-verify`. The hooks re-point `core.hooksPath` at a *relative*
`.githooks` on every run, so a worktree runs its own branch's hooks rather than
the main checkout's.

The tag has a second use, same mechanism and a different reason: **production code that is written
and tested but has no caller yet**. `clicks.Geography`'s `Disc` is there: the bomb used it for a
release, then moved to a circle. The tag is what keeps `make deadcode` a wall rather than a thing people learn to
ignore, and removing the line is the whole of promoting such a function.

**`make deadcode` fails on any unreachable function**, in two passes, because "is this reachable?" has two different right answers depending on whether test code counts as a caller. The first pass excludes tests and tagged files, so **production code whose only caller is a test is reported as dead** — the case a plain `deadcode -test` forgives. The second pass includes both but keeps only findings inside tagged files, so an unused shared helper is reported too. `deadcode` is fetched at a pinned version by the target, so there is nothing to install.
