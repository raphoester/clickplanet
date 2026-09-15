package shadowban

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// The pre-postgres bans file, read once to import it. Delete this file once production runs on postgres.

const importedSuffix = ".imported"

type legacyBan struct {
	Scope    string    `json:"scope"`
	Flags    int       `json:"flags"`
	Offences int       `json:"offences"`
	Until    time.Time `json:"until"`
}

// importLegacyStateLocked refuses a file it cannot read: starting without its bans unbans every bot in it.
func (b *Banner) importLegacyStateLocked() error {
	path := b.config.LegacyStatePath
	if !legacyFileExists(path) {
		return nil
	}

	//nolint:gosec // G304: the path comes from config, never from a request.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read the legacy bans file %s: %w", path, err)
	}

	bans, err := decodeLegacyBans(data)
	if err != nil {
		return fmt.Errorf("failed to import the legacy bans file %s, move it away to start with no bans: %w", path, err)
	}

	for scope, record := range bans {
		b.bans[scope] = record
		b.dirty[scope] = struct{}{}
	}

	b.imported = path
	if len(bans) == 0 {
		b.retireLegacyStateLocked()
	}

	return nil
}

func decodeLegacyBans(data []byte) (map[string]*ban, error) {
	bans := make(map[string]*ban)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for line := 1; scanner.Scan(); line++ {
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}

		var saved legacyBan
		if err := json.Unmarshal(scanner.Bytes(), &saved); err != nil || saved.Scope == "" {
			return nil, fmt.Errorf("corrupt ban on line %d", line)
		}

		bans[saved.Scope] = Record(saved).ban()
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan: %w", err)
	}

	return bans, nil
}

// retireLegacyState renames the imported file once postgres holds it, so a later boot cannot import it again.
func (b *Banner) retireLegacyState() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.retireLegacyStateLocked()
}

func (b *Banner) retireLegacyStateLocked() {
	if b.imported == "" {
		return
	}

	path := b.imported
	b.imported = ""

	if err := os.Rename(path, path+importedSuffix); err != nil {
		b.onStateError(fmt.Errorf("imported the legacy bans file but failed to rename it: %w", err))
	}
}

func legacyFileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}
