package migrations

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Pattern is where each module keeps the migrations it embeds, from the backend's root.
const Pattern = "internal/*/internal/migrations/*.sql"

var fileName = regexp.MustCompile(`^([0-9]{14})_([a-z0-9_]+)\.(up|down)\.sql$`)

const versionLayout = "20060102150405"

type Version uint64

type Problem struct {
	Directory string
	Reason    string
}

// Tree is the migration file names of each module, by the directory that holds them.
type Tree struct {
	files map[string][]string
}

func NewTree(paths []string) Tree {
	files := map[string][]string{}
	for _, p := range paths {
		if matched, _ := path.Match(Pattern, p); !matched {
			continue
		}
		dir, name := path.Split(p)
		dir = strings.TrimSuffix(dir, "/")
		files[dir] = append(files[dir], name)
	}
	return Tree{files: files}
}

func TreeIn(fsys fs.FS) (Tree, error) {
	paths, err := fs.Glob(fsys, Pattern)
	if err != nil {
		return Tree{}, fmt.Errorf("failed to list the migrations: %w", err)
	}
	return NewTree(paths), nil
}

// TreeAt lists the migrations of a git revision. Run from the backend's root: git lists paths from there.
func TreeAt(ctx context.Context, revision string) (Tree, error) {
	out, err := exec.CommandContext(ctx, "git", "ls-tree", "-r", "--name-only", "--end-of-options", revision, "--", "internal").Output() //nolint:gosec // --end-of-options keeps the revision an argument.
	if err != nil {
		if exited, ok := errors.AsType[*exec.ExitError](err); ok {
			return Tree{}, fmt.Errorf("failed to list the migrations at %s: %w: %s", revision, err, strings.TrimSpace(string(exited.Stderr)))
		}
		return Tree{}, fmt.Errorf("failed to list the migrations at %s: %w", revision, err)
	}
	return NewTree(strings.Split(strings.TrimSpace(string(out)), "\n")), nil
}

// Problems are what would make golang-migrate refuse to start, or skip a migration, on a database that ran every
// migration of ran and then boots head.
func Problems(ran, head Tree) []Problem {
	var problems []Problem
	for _, dir := range slices.Sorted(maps.Keys(head.files)) {
		migrations, found := directoryOf(dir, head.files[dir])
		problems = append(problems, found...)
		before, ok := ran.files[dir]
		if !ok {
			continue
		}
		previous, _ := directoryOf(dir, before)
		problems = append(problems, previous.orderProblems(migrations)...)
	}
	return problems
}

type migration struct {
	files []string
	names []string
	up    bool
}

type directory struct {
	path       string
	migrations map[Version]*migration
}

func directoryOf(dir string, files []string) (directory, []Problem) {
	parsed := directory{path: dir, migrations: map[Version]*migration{}}
	var problems []Problem
	for _, file := range slices.Sorted(slices.Values(files)) {
		parts := fileName.FindStringSubmatch(file)
		if parts == nil {
			problems = append(problems, parsed.problem("%s is not <yyyymmddhhmmss>_<snake_case>.up.sql or .down.sql: golang-migrate skips it, or runs it out of order", file))
			continue
		}
		if _, err := time.Parse(versionLayout, parts[1]); err != nil {
			problems = append(problems, parsed.problem("%s: %s is not a UTC date and time, yyyymmddhhmmss", file, parts[1]))
			continue
		}
		number, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			problems = append(problems, parsed.problem("%s: %s is not a version", file, parts[1]))
			continue
		}
		m, ok := parsed.migrations[Version(number)]
		if !ok {
			m = &migration{}
			parsed.migrations[Version(number)] = m
		}
		m.files = append(m.files, file)
		if !slices.Contains(m.names, parts[2]) {
			m.names = append(m.names, parts[2])
		}
		if parts[3] == "up" {
			m.up = true
		}
	}
	for _, version := range parsed.versions() {
		m := parsed.migrations[version]
		if len(m.names) > 1 {
			problems = append(problems, parsed.problem("version %d is %d migrations (%s): golang-migrate refuses to start. Give each a version of its own", version, len(m.names), strings.Join(m.files, ", ")))
		}
		if !m.up {
			problems = append(problems, parsed.problem("version %d has no .up.sql (%s)", version, strings.Join(m.files, ", ")))
		}
	}
	return parsed, problems
}

func (d directory) orderProblems(head directory) []Problem {
	versions := d.versions()
	if len(versions) == 0 {
		return nil
	}
	newest := versions[len(versions)-1]
	var problems []Problem
	if _, ok := head.migrations[newest]; !ok {
		problems = append(problems, d.problem("%s is gone, and a database that ran it does not start: golang-migrate looks for the version it is at", d.nameOf(newest)))
	}
	for _, version := range head.versions() {
		if _, ran := d.migrations[version]; ran || version > newest {
			continue
		}
		problems = append(problems, d.problem("%s is new, but older than %s, which already ran: golang-migrate only runs what comes after, and would skip it. Give it a version after %d", head.nameOf(version), d.nameOf(newest), newest))
	}
	return problems
}

func (d directory) versions() []Version {
	return slices.Sorted(maps.Keys(d.migrations))
}

func (d directory) nameOf(version Version) string {
	return fmt.Sprintf("%d_%s", version, d.migrations[version].names[0])
}

func (d directory) problem(format string, args ...any) Problem {
	return Problem{Directory: d.path, Reason: fmt.Sprintf(format, args...)}
}
