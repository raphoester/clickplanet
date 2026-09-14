package shadowban

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomicfile"
)

// One JSON object per line, so a single scope can be unbanned with grep -v.
type savedBan struct {
	Scope     string    `json:"scope"`
	Flags     int       `json:"flags"`
	Offences  int       `json:"offences"`
	Until     time.Time `json:"until,omitzero"`
	Permanent bool      `json:"permanent,omitempty"`
}

func (b *Banner) restore() error {
	if b.config.StatePath == "" {
		return nil
	}

	//nolint:gosec // G304: the path comes from config, never from a request.
	data, err := os.ReadFile(b.config.StatePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read bans from %s: %w", b.config.StatePath, err)
	}

	bans := make(map[string]*ban)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for line := 1; scanner.Scan(); line++ {
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}

		var saved savedBan
		if err := json.Unmarshal(scanner.Bytes(), &saved); err != nil || saved.Scope == "" {
			return fmt.Errorf("corrupt ban on line %d of %s, starting with no bans", line, b.config.StatePath)
		}

		bans[saved.Scope] = &ban{
			flags:     saved.Flags,
			offences:  saved.Offences,
			until:     saved.Until,
			permanent: saved.Permanent,
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read bans from %s: %w", b.config.StatePath, err)
	}

	b.mu.Lock()
	b.bans = bans
	b.mu.Unlock()

	return nil
}

func (b *Banner) saveIfDirty() {
	if b.config.StatePath == "" {
		return
	}

	b.mu.Lock()
	if !b.dirty {
		b.mu.Unlock()
		return
	}
	b.dirty = false

	saved := make([]savedBan, 0, len(b.bans))
	for scope, record := range b.bans {
		saved = append(saved, savedBan{
			Scope:     scope,
			Flags:     record.flags,
			Offences:  record.offences,
			Until:     record.until,
			Permanent: record.permanent,
		})
	}
	b.mu.Unlock()

	sort.Slice(saved, func(i, j int) bool { return saved[i].Scope < saved[j].Scope })

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, record := range saved {
		if err := encoder.Encode(record); err != nil {
			b.markDirty()
			b.onStateError(fmt.Errorf("failed to encode bans: %w", err))
			return
		}
	}

	if err := cpatomicfile.Write(b.config.StatePath, buf.Bytes()); err != nil {
		b.markDirty()
		b.onStateError(fmt.Errorf("failed to save bans to %s: %w", b.config.StatePath, err))
	}
}

func (b *Banner) markDirty() {
	b.mu.Lock()
	b.dirty = true
	b.mu.Unlock()
}
