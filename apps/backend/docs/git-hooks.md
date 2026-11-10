# Git hooks

`./.githooks/install` points git at [`.githooks/`](../../../.githooks). It is a
script rather than a root `Makefile` target because this repo keeps build tooling
inside each app, and hooks are the one genuinely repo-wide thing.

- **pre-commit** — `make tidy` (re-staging only what was already staged, plus
  `go.mod`/`go.sum` if tidy moved them) and `make lint-ci`, plus the frontend's
  and the backoffice's eslint, each only when that app has staged changes.
- **commit-msg** — conventional commits, which the history already uses, and a
  72-character subject so `git log --oneline` stays readable. Merge, revert,
  fixup and squash subjects are git's to format and are left alone.
- **pre-push** — `make test`, `make deadcode`, `make lint-ci`,
  `make check-format`, `make check-quiz` and `make check-moderation`, run
  concurrently; output is only printed for a step that fails.

All three take `--no-verify`. The hooks re-point `core.hooksPath` at a *relative*
`.githooks` on every run, so a worktree runs its own branch's hooks rather than
the main checkout's.
