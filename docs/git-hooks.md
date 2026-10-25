# Git hooks

`./.githooks/install` points git at [`.githooks/`](../.githooks), once per clone.
pre-commit formats (gofumpt, via `make tidy`) and lints whichever app has staged
changes, commit-msg enforces conventional commits, and pre-push runs the
backend's tests, dead-code check, linter and format check concurrently, and checks
that the backend's copy of the quiz bank is the one in `quiz/`. All three
take `--no-verify`.

It is a script rather than a root `Makefile` target on purpose — see
[Independence of the apps](app-independence.md).
