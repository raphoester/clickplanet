# Linting

`make lint` runs **golangci-lint**, configured in [`.golangci.yaml`](../.golangci.yaml).
The version is pinned in the `Makefile` and matched by the CI job, because an
unpinned linter turns a green branch red on somebody else's release schedule.
`make setup-tools` installs that version.

**Formatting is gofumpt**, enabled in the `formatters` block of the same file
and run by `make tidy`. It goes through golangci-lint rather than a `gofumpt`
binary of its own: one pinned version to install instead of two that can
disagree, and it already knows which files are generated. gofumpt is a strict
superset of gofmt, so anything it accepts `gofmt` accepts.

`make check-format` is the non-rewriting half, and it is what CI runs. Note that
`golangci-lint fmt --diff` **prints a diff but exits 0 either way**, so the
target tests its output rather than its status — a `check-format` written the
obvious way passes on an unformatted tree. It also runs `go mod tidy` and fails
on a resulting diff, so an untidy `go.mod` is caught in the same place.

**`run.build-tags` is `testing`**, for the same reason `go vet` needs it: without
the tag the linters load a tree where the shared helpers are undefined and report
that instead of anything real.

**Test files are linted like production code.** The only per-path exclusion is
`generated/`, which `make proto` owns. Where a linter is wrong about a specific
line, the line carries a `//nolint` naming the linter and the reason — which
keeps the rule live everywhere else:

- `nilnil` — three constructors return `(nil, nil)` for **"this feature is off"**,
  and the caller checks for nil and mounts nothing. A sentinel would make every
  caller unwrap one. Annotated per site so an *accidental* `nil, nil` is still caught.
- `gosec` G404 — `math/rand` in the antibot tests is deterministic on purpose:
  a fixed seed replays the exact click stream a watchdog is asserted against.
- `bodyclose` — it cannot see a body closed by a helper's `t.Cleanup`.

**Two linters are deliberately off**, and both were switched off on evidence
rather than left out:

- **`exhaustruct`** wants every field named at every struct literal. That suits
  DTOs; it does not suit this codebase, where most structs are stateful objects
  whose zero values are correct and meaningful. It produced 161 findings, of
  which the representative ones are `Jury is missing field mu` (a `sync.Mutex`
  you cannot meaningfully write in a literal), `caller is missing fields clicks,
  tiles` (counters that start at zero), and nine `Evidence is missing fields
  Rule, Fields` — where `Evidence{}` *is* the "nothing to report" value.
- **`forbidigo`** has nothing to forbid here yet. It is worth turning on the day
  a retired pattern needs to stay retired.

`wrapcheck` is on, with `extra-ignore-sigs` for the signatures this codebase
returns bare **on purpose**: use cases called by a handler, because the handler maps the
domain sentinels centrally and a wrap would put a second sentence in front of a
message it already chose; and pure delegations, where the callee already named
what failed.
