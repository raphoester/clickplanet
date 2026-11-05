# A command's `run` is a table of contents

**Every CLI tool splits its `main` into named stages with one job each.** The top-level `run` names the stages and wires them together, and does nothing else:

```go
// Good
func run(ctx context.Context, logger *slog.Logger) error {
    flags, err := parseFlags()
    if err != nil {
        return err
    }
    results, err := doWork(ctx, flags, ...)
    if err != nil {
        return err
    }
    ok := displayResults(results)
    if !ok {
        return fmt.Errorf("one or more checks failed")
    }
    return nil
}

// Bad: one run that parses, executes and prints inline
func run(logger *slog.Logger) error {
    servicePath := flag.String(...)
    flag.Parse()
    // ... business logic ...
    // ... printing ...
    anyFailed := false
    // ...
}
```

- **`parseFlags` returns a typed config struct** (`FlagsConfig`). A raw `*string` flag value is never an argument of a function downstream.
- **A work function returns values.** It does not write into a variable of an outer scope. The body of a goroutine is ideally one assignment: `results[i] = doOneUnit(...)`.
- **A display function returns a `bool`, or a status.** It owns the output, not the propagation of errors: the caller decides what to do with what it answers.
- **A signature is honest**: a function that can fail returns an `error`, and one that cannot does not.
- **Package `main` stays thin: `main`, `run`, and nothing that does the work of a stage.** The stages live in packages of their own. A parser or a table that sits in `main` beside `run` is the monolith again, in three files.
- **A dispatch names every value it acts on, and anything else is an error.** No `default:` that runs a command: a value nobody expected then does the last thing in the list, and here that is posting.

**`cmd/clips` follows it** (see [The clips job](clips-job.md)): `flags.Parse` in `cmd/clips/internal/flags`, the work in `internal/clips/job.go`, `display.Picks` and `display.Report` in `cmd/clips/internal/display`. `main.go` is `main`, `run`, the dispatch and one short function per command. What it settled:

- **With subcommands, `run` parses, then picks the command**, and each command is its own short list of stages: work, then display. `stats` has no display: what it keeps is read in the backoffice, so it is work, then the exit status. The dispatch has a `case` per command and fails on any other (`errUnknownCommand`), though the parser already refused it.
- **The parser and the display are the tool's own, under `cmd/clips/internal/`**: Go keeps them out of reach of everything but this command, and they import the module, never the reverse.
- **A display that writes can fail**, so it answers an `error` beside its `bool`: `display.Report(summaryFile, out, report) (bool, error)`. The `bool` is still what the report says (every clip is scheduled), and the `error` only that the summary could not be written.

`cmd/api` is a server, not a tool: its `run` loads the config and hands the modules to `cpbootstrap`.
