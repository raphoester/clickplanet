package file_stats_store

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/stats/usecases/take_snapshot_usecase"
)

const (
	snapshots = "snapshots"
	table     = "clips-stats.csv"
	// No colon: the folder is a git repository, and a colon is no file name on Windows.
	stampLayout = "2006-01-02T15-04-05Z"
)

// A folder: snapshots/<time>.jsonl, a row of JSON a line, for each snapshot, the source of truth; and clips-stats.csv,
// the table of every row of every snapshot, made again from them each time one is kept.
type Store string

var _ take_snapshot_usecase.Store = Store("")

func (s Store) Keep(_ context.Context, takenAt time.Time, rows []stats.Row) error {
	folder := filepath.Join(string(s), snapshots)
	if err := os.MkdirAll(folder, 0o750); err != nil {
		return fmt.Errorf("failed to make the folder of the snapshots: %w", err)
	}

	if len(rows) > 0 {
		var lines bytes.Buffer
		encoder := json.NewEncoder(&lines)
		encoder.SetEscapeHTML(false)
		for _, row := range rows {
			if err := encoder.Encode(row); err != nil {
				return fmt.Errorf("failed to write a row of the snapshot: %w", err)
			}
		}
		file := filepath.Join(folder, takenAt.UTC().Format(stampLayout)+".jsonl")
		if err := os.WriteFile(file, lines.Bytes(), 0o600); err != nil {
			return fmt.Errorf("failed to write the snapshot: %w", err)
		}
	}

	all, err := s.Rows()
	if err != nil {
		return err
	}
	return s.writeTable(all)
}

// Every row of every snapshot, the oldest snapshot first, each in the order it was written.
func (s Store) Rows() ([]stats.Row, error) {
	files, err := filepath.Glob(filepath.Join(string(s), snapshots, "*.jsonl"))
	if err != nil {
		return nil, fmt.Errorf("failed to list the snapshots: %w", err)
	}
	sort.Strings(files)

	var rows []stats.Row
	for _, file := range files {
		read, err := rowsOf(file)
		if err != nil {
			return nil, err
		}
		rows = append(rows, read...)
	}
	return rows, nil
}

func rowsOf(file string) ([]stats.Row, error) {
	content, err := os.ReadFile(file) //nolint:gosec // a snapshot of the folder the operator of the job named.
	if err != nil {
		return nil, fmt.Errorf("failed to read the snapshot %s: %w", filepath.Base(file), err)
	}
	var rows []stats.Row
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(nil, 16<<20)
	for line := 1; scanner.Scan(); line++ {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var row stats.Row
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			return nil, fmt.Errorf("line %d of the snapshot %s is not a row: %w", line, filepath.Base(file), err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read the snapshot %s: %w", filepath.Base(file), err)
	}
	return rows, nil
}

func (s Store) writeTable(rows []stats.Row) error {
	header, records := stats.Table(rows)

	var written bytes.Buffer
	writer := csv.NewWriter(&written)
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write the table: %w", err)
	}
	if err := writer.WriteAll(records); err != nil {
		return fmt.Errorf("failed to write the table: %w", err)
	}
	if err := os.WriteFile(filepath.Join(string(s), table), written.Bytes(), 0o600); err != nil {
		return fmt.Errorf("failed to write the table: %w", err)
	}
	return nil
}
