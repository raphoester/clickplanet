package memory_chat_storage

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

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomicfile"
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

func toLogRecord(record domain.ChatRecord) logRecord {
	return logRecord{
		At:        record.Message.SentAt.UTC(),
		ID:        record.Message.ID,
		Name:      record.Message.AuthorName,
		Tag:       record.Message.AuthorTag,
		AuthorID:  record.AuthorID,
		Country:   record.Message.CountryID,
		IP:        record.IP,
		UserAgent: record.UserAgent,
		Text:      record.Message.Text,
	}
}

func (r logRecord) toMessage() domain.ChatMessage {
	return domain.ChatMessage{
		ID:         r.ID,
		SentAt:     r.At,
		AuthorName: r.Name,
		AuthorTag:  r.Tag,
		CountryID:  r.Country,
		Text:       r.Text,
	}
}

type appendLog struct {
	file  *os.File
	dirty bool
}

func (s *Storage) Run(ctx context.Context) {
	if s.config.LogPath == "" {
		s.logger.Warn("no chat log path configured, messages will not survive a restart")
		<-ctx.Done()
		return
	}

	s.logger.Info("logging chat",
		slog.String("path", s.config.LogPath),
		slog.Duration("retention", s.config.Retention),
	)

	flush := time.NewTicker(s.config.FlushInterval)
	defer flush.Stop()

	prune := time.NewTicker(s.config.PruneInterval)
	defer prune.Stop()

	for {
		select {
		case <-flush.C:
			s.flush()
		case <-prune.C:
			s.prune()
		case <-ctx.Done():
			s.flush()
			s.closeLog()
			return
		}
	}
}

func (s *Storage) restore() {
	if s.config.LogPath == "" {
		return
	}

	if err := cpatomicfile.CheckWritable(s.config.LogPath); err != nil {
		s.logger.Error("chat log path is not writable, messages will not be recorded",
			slog.String("path", s.config.LogPath),
			slog.Any("error", err),
		)
		return
	}

	records, skipped, err := readRecords(s.config.LogPath)
	if err != nil {
		s.logger.Error("failed to read the chat log, starting with an empty history",
			slog.String("path", s.config.LogPath),
			slog.Any("error", err),
		)
	}
	if skipped > 0 {
		s.logger.Warn("skipped unreadable chat log lines",
			slog.String("path", s.config.LogPath),
			slog.Int("skipped", skipped),
		)
	}

	kept := withinRetention(records, s.clock.Now().Add(-s.config.Retention))
	if len(kept) > s.config.HistorySize {
		kept = kept[len(kept)-s.config.HistorySize:]
	}

	s.history = s.history[:0]
	for _, record := range kept {
		s.history = append(s.history, record.toMessage())
	}

	file, err := openForAppend(s.config.LogPath)
	if err != nil {
		s.logger.Error("failed to open the chat log for appending",
			slog.String("path", s.config.LogPath),
			slog.Any("error", err),
		)
		return
	}

	s.log = &appendLog{file: file}
}

func (s *Storage) appendToLog(record domain.ChatRecord) error {
	s.logMu.Lock()
	defer s.logMu.Unlock()

	if s.log == nil {
		return nil
	}

	line, err := json.Marshal(toLogRecord(record))
	if err != nil {
		return fmt.Errorf("failed to encode the record: %w", err)
	}

	if _, err := s.log.file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("failed to append: %w", err)
	}

	s.log.dirty = true
	return nil
}

func (s *Storage) flush() {
	s.logMu.Lock()
	defer s.logMu.Unlock()

	if s.log == nil || !s.log.dirty {
		return
	}

	if err := s.log.file.Sync(); err != nil {
		s.logger.Error("failed to flush the chat log",
			slog.String("path", s.config.LogPath),
			slog.Any("error", err),
		)
		return
	}

	s.log.dirty = false
}

func (s *Storage) closeLog() {
	s.logMu.Lock()
	defer s.logMu.Unlock()

	if s.log == nil {
		return
	}

	if err := s.log.file.Close(); err != nil {
		s.logger.Error("failed to close the chat log", slog.Any("error", err))
	}

	s.log = nil
}

func (s *Storage) prune() {
	s.logMu.Lock()
	defer s.logMu.Unlock()

	if s.log == nil {
		return
	}

	if err := s.log.file.Sync(); err != nil {
		s.logger.Error("failed to flush the chat log before pruning", slog.Any("error", err))
		return
	}
	s.log.dirty = false

	records, _, err := readRecords(s.config.LogPath)
	if err != nil {
		s.logger.Error("failed to read the chat log for pruning", slog.Any("error", err))
		return
	}

	kept := withinRetention(records, s.clock.Now().Add(-s.config.Retention))
	if len(kept) == len(records) {
		return
	}

	var payload bytes.Buffer
	for _, record := range kept {
		line, err := json.Marshal(record)
		if err != nil {
			s.logger.Error("failed to re-encode a chat record while pruning", slog.Any("error", err))
			return
		}
		payload.Write(line)
		payload.WriteByte('\n')
	}

	if err := s.log.file.Close(); err != nil {
		s.logger.Error("failed to close the chat log before pruning", slog.Any("error", err))
	}
	s.log = nil

	if err := cpatomicfile.Write(s.config.LogPath, payload.Bytes()); err != nil {
		s.logger.Error("failed to write the pruned chat log", slog.Any("error", err))
	}

	file, err := openForAppend(s.config.LogPath)
	if err != nil {
		s.logger.Error("failed to reopen the chat log after pruning", slog.Any("error", err))
		return
	}

	s.log = &appendLog{file: file}
	s.logger.Info("pruned the chat log",
		slog.Int("dropped", len(records)-len(kept)),
		slog.Int("kept", len(kept)),
	)
}

func openForAppend(path string) (*os.File, error) {
	//nolint:gosec // G304: path is chat.storage.logPath from config, never a request.
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %w", path, err)
	}
	return file, nil
}

func readRecords(path string) ([]logRecord, int, error) {
	//nolint:gosec // G304: path is chat.storage.logPath from config, never a request.
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("failed to open %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	var (
		records []logRecord
		skipped int
	)

	scanner := bufio.NewScanner(file)
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

		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return records, skipped, fmt.Errorf("failed to read %q: %w", path, err)
	}

	return records, skipped, nil
}

func withinRetention(records []logRecord, cutoff time.Time) []logRecord {
	kept := make([]logRecord, 0, len(records))
	for _, record := range records {
		if record.At.Before(cutoff) {
			continue
		}
		kept = append(kept, record)
	}
	return kept
}
