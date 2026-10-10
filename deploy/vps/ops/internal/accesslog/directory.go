package accesslog

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-ops/internal/excerpt"
)

func NewDirectory(path string) *Directory {
	return &Directory{path: path}
}

type Directory struct {
	path string
}

// Caddy zips a rotated file a moment after it opens the next one.
const rotationSlack = time.Minute

type logFile struct {
	path     string
	lastLine time.Time
}

func (d *Directory) Read(ctx context.Context, query Query) (excerpt.Excerpt, error) {
	files, err := d.files()
	if err != nil {
		return excerpt.Excerpt{}, err
	}

	collector := excerpt.NewCollector(query.Limit)
	keep := kept(query)

	var firstLine time.Time
	for _, file := range files {
		covers := !file.lastLine.Before(query.Since) && !firstLine.After(query.Until)
		firstLine = file.lastLine.Add(-rotationSlack)
		if !covers {
			continue
		}
		if err := ctx.Err(); err != nil {
			return excerpt.Excerpt{}, fmt.Errorf("stopped before %s: %w", filepath.Base(file.path), err)
		}
		if err := fill(collector, file.path, keep); err != nil {
			return excerpt.Excerpt{}, err
		}
		if collector.Full() {
			break
		}
	}
	return collector.Excerpt(), nil
}

// Oldest first. Caddy stamps a file when it writes to it, so each one ends where the next begins.
func (d *Directory) files() ([]logFile, error) {
	entries, err := os.ReadDir(d.path)
	if err != nil {
		return nil, fmt.Errorf("failed to list the access logs: %w", err)
	}

	files := make([]logFile, 0, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("failed to stat %s: %w", entry.Name(), err)
		}
		files = append(files, logFile{path: filepath.Join(d.path, entry.Name()), lastLine: info.ModTime()})
	}
	slices.SortFunc(files, func(a, b logFile) int { return a.lastLine.Compare(b.lastLine) })
	return files, nil
}

func kept(query Query) func(line []byte) bool {
	wanted := []byte(query.Contains)
	since := float64(query.Since.UnixNano()) / float64(time.Second)
	until := float64(query.Until.UnixNano()) / float64(time.Second)

	return func(line []byte) bool {
		if !bytes.Contains(line, wanted) {
			return false
		}
		var entry struct {
			Timestamp float64 `json:"ts"`
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			return false
		}
		return entry.Timestamp >= since && entry.Timestamp <= until
	}
}

func fill(collector *excerpt.Collector, path string, keep func(line []byte) bool) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", filepath.Base(path), err)
	}
	defer func() { _ = file.Close() }()

	var lines io.Reader = file
	if strings.HasSuffix(path, ".gz") {
		unzipped, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to unzip %s: %w", filepath.Base(path), err)
		}
		defer func() { _ = unzipped.Close() }()
		lines = unzipped
	}

	if err := collector.Fill(lines, keep); err != nil {
		return fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
	}
	return nil
}
