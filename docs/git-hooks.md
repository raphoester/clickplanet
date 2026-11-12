# Git hooks

`./.githooks/install` points git at [`.githooks/`](../.githooks), once per clone.
pre-commit formats (gofumpt, via `make tidy`) and lints whichever app has staged
changes, commit-msg enforces conventional commits, and pre-push runs the
backend's tests, dead-code check, linter and format check concurrently, and checks
that the backend's copies of the quiz bank and of the moderation policy are the
ones in `quiz/` and `docs/`. All three take `--no-verify`.

pre-push runs its checks only when a pushed commit changes what they read:
`apps/backend/`, `proto/`, `quiz/`, `docs/moderation.md`, and `deploy/`, whose
config files the backend's tests load. Otherwise it says the checks were skipped.
A new branch is compared with `origin/main`. When the hook cannot tell what
changed (a remote commit it does not have, no `origin/main`), it runs them.
CI's `check backend` workflow runs on the same paths: a change to one list is a
change to the other.

It is a script rather than a root `Makefile` target on purpose — see
[Independence of the apps](app-independence.md).
