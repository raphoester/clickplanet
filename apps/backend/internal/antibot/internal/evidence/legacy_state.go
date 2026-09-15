package evidence

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"hash/crc32"
	"io/fs"
	"os"
)

// The pre-postgres evidence file, read once to import it. Delete this file once production runs on postgres.

// Magic, version and a CRC32 of the payload, then the payload as gob.
const (
	stateMagic   = "CPEVIDN\n"
	stateVersion = uint8(1)
	headerSize   = len(stateMagic) + 1 + 4
)

const importedSuffix = ".imported"

var errCorruptState = errors.New("corrupt antibot evidence file")

type legacyFile struct {
	SavedAt  int64
	Sections map[string][]byte
}

// importLegacyState refuses a file it cannot read, so an operator decides to lose it rather than a boot.
func (s *Store) importLegacyState() (Snapshot, error) {
	path := s.config.LegacyStatePath
	if !legacyFileExists(path) {
		return Snapshot{}, nil
	}

	//nolint:gosec // G304: the path comes from config, never from a request.
	raw, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("failed to read the legacy evidence file %s: %w", path, err)
	}

	saved, err := decodeLegacyFile(raw)
	if err != nil {
		return Snapshot{}, fmt.Errorf("failed to import the legacy evidence file %s, move it away to start empty: %w", path, err)
	}

	s.mu.Lock()
	s.imported = path
	s.mu.Unlock()

	return Snapshot{SavedAt: Time(saved.SavedAt), Sections: saved.Sections}, nil
}

func decodeLegacyFile(raw []byte) (legacyFile, error) {
	if len(raw) < headerSize {
		return legacyFile{}, fmt.Errorf("%w: file is %d bytes, shorter than the header", errCorruptState, len(raw))
	}
	if string(raw[:len(stateMagic)]) != stateMagic {
		return legacyFile{}, fmt.Errorf("%w: bad magic", errCorruptState)
	}
	if version := raw[len(stateMagic)]; version != stateVersion {
		return legacyFile{}, fmt.Errorf("%w: unsupported version %d", errCorruptState, version)
	}

	payload := raw[headerSize:]
	want := binary.LittleEndian.Uint32(raw[len(stateMagic)+1:])
	if got := crc32.ChecksumIEEE(payload); got != want {
		return legacyFile{}, fmt.Errorf("%w: checksum mismatch (want %08x, got %08x)", errCorruptState, want, got)
	}

	var saved legacyFile
	if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(&saved); err != nil {
		return legacyFile{}, fmt.Errorf("%w: %w", errCorruptState, err)
	}

	return saved, nil
}

// retireLegacyState renames the imported file once postgres holds it, so a later boot cannot import it again.
func (s *Store) retireLegacyState() {
	s.mu.Lock()
	path := s.imported
	s.imported = ""
	s.mu.Unlock()

	if path == "" {
		return
	}

	if err := os.Rename(path, path+importedSuffix); err != nil {
		s.onStateError(fmt.Errorf("imported the legacy evidence file but failed to rename it: %w", err))
	}
}

func legacyFileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return !errors.Is(err, fs.ErrNotExist)
}
