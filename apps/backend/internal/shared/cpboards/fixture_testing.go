//go:build testing

package cpboards

import (
	"embed"
	"errors"
	"io/fs"
	"slices"
	"strings"
)

// Fixture is the tests' second board: a coarse earth cut from earth-1 at cols 31, its tiles about 9.7 times as far
// apart and a hundredth as dense, with another positions hash. The page's `npm run map:fixture` writes it, into each
// app's test data alone: no app plays it, and planet keeps a row of it only in a test's database.
const Fixture = "fixture-1"

//go:embed testdata
var fixtures embed.FS

// withFixture is the boards, and the fixture beside them.
func withFixture(boards fs.FS) fs.FS {
	fixture, err := fs.Sub(fixtures, "testdata")
	if err != nil {
		panic(err) // a constant path.
	}
	return union{boards, fixture}
}

// FixtureBoard is the fixture, read as any board.
func FixtureBoard() Board {
	board, err := Named(Files(), Fixture)
	if err != nil {
		panic(err)
	}
	return board
}

// union reads each path from the first file system that has it, and lists the top directory of both.
type union []fs.FS

func (u union) Open(name string) (fs.File, error) {
	for _, files := range u {
		file, err := files.Open(name)
		if err == nil {
			return file, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err //nolint:wrapcheck // the file system names it.
		}
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

func (u union) ReadDir(name string) ([]fs.DirEntry, error) {
	var all []fs.DirEntry
	found := false
	for _, files := range u {
		entries, err := fs.ReadDir(files, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err //nolint:wrapcheck // the file system names it.
		}
		found = true
		all = append(all, entries...)
	}
	if !found {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	slices.SortFunc(all, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return all, nil
}
