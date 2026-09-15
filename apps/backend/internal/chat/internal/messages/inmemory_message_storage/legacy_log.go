package inmemory_message_storage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
)

// The pre-postgres JSONL log, read once to import it. Delete this file once production runs on postgres.

const (
	importedSuffix = ".imported"
	maxLineSize    = 1 << 20
)

type logRecord struct {
	At        time.Time `json:"at"`
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Tag       string    `json:"tag"`
	AuthorID  string    `json:"authorId,omitempty"`
	Country   string    `json:"country"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent,omitempty"`
	Text      string    `json:"text"`
}

func (r logRecord) toRecord() messages.Record {
	return messages.Record{
		Message: messages.Message{
			ID:         r.ID,
			SentAt:     r.At,
			AuthorName: r.Name,
			AuthorTag:  r.Tag,
			CountryID:  r.Country,
			Text:       r.Text,
		},
		AuthorID:  r.AuthorID,
		IP:        r.IP,
		UserAgent: r.UserAgent,
	}
}

// importLegacyLog refuses a log it cannot read: once a message lands in the empty table it would never be imported.
func (s *Storage) importLegacyLog(ctx context.Context) error {
	path := s.config.LegacyLogPath
	if path == "" || !fileExists(path) {
		return nil
	}

	empty, err := s.persistence.IsEmpty(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for stored chat messages: %w", err)
	}
	if !empty {
		s.logger.Warn("a legacy chat log is still on disk but postgres already holds messages, ignoring it",
			slog.String("path", path))
		return nil
	}

	records, skipped, err := readLegacyLog(path)
	if err != nil {
		return fmt.Errorf("failed to import the legacy chat log %s, move it away to start empty: %w", path, err)
	}
	if skipped > 0 {
		s.logger.Warn("skipped unreadable chat log lines", slog.String("path", path), slog.Int("skipped", skipped))
	}

	kept := withinRetention(records, s.cutoff())
	if err := s.persistence.InsertAll(ctx, kept); err != nil {
		return fmt.Errorf("failed to import the legacy chat log %s: %w", path, err)
	}

	s.logger.Info("imported the legacy chat log",
		slog.String("path", path),
		slog.Int("imported", len(kept)),
		slog.Int("pastRetention", len(records)-len(kept)),
	)

	if err := os.Rename(path, path+importedSuffix); err != nil {
		s.logger.Error("failed to rename the imported chat log", slog.String("path", path), slog.Any("error", err))
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}

func readLegacyLog(path string) ([]messages.Record, int, error) {
	//nolint:gosec // G304: path is chat.storage.legacyLogPath from config, never a request.
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	var (
		records []messages.Record
		skipped int
	)

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxLineSize)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var record logRecord
		if err := json.Unmarshal(line, &record); err != nil {
			skipped++
			continue
		}

		records = append(records, record.toRecord())
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to read %q: %w", path, err)
	}

	return records, skipped, nil
}

func withinRetention(records []messages.Record, cutoff time.Time) []messages.Record {
	kept := make([]messages.Record, 0, len(records))
	for _, record := range records {
		if record.Message.SentAt.Before(cutoff) {
			continue
		}
		kept = append(kept, record)
	}
	return kept
}
