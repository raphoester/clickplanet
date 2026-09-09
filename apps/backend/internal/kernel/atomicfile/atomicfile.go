// Package atomicfile writes a file in a way that survives a crash or a power
// loss mid-write: the previous contents are either fully replaced or left
// untouched, never half-overwritten.
//
// It started life inside the tile storage's snapshot code and moved here when
// the chat log needed the same guarantee for its retention rewrites.
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// Write writes data to a temp file in the destination directory, fsyncs it,
// then renames it over path. A crash mid-write leaves the previous contents
// intact.
func Write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()

	defer func() {
		// No-op once the rename succeeded, cleanup otherwise.
		_ = os.Remove(tmpName)
	}()

	// CreateTemp makes the file 0600; nothing written through here is secret
	// and backup jobs may well run as another user.
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to chmod temp file: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return syncDir(dir)
}

// syncDir flushes the rename itself, so the new contents survive a power loss
// and not just a process crash.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}
	defer func() { _ = d.Close() }()

	if err := d.Sync(); err != nil {
		return fmt.Errorf("failed to sync directory: %w", err)
	}

	return nil
}

// CheckWritable reports whether a file could be written to path, without
// disturbing whatever is already there. Callers use it to surface an
// unwritable destination at startup rather than at the first write.
func CheckWritable(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	probe, err := os.CreateTemp(dir, filepath.Base(path)+".probe-*")
	if err != nil {
		return fmt.Errorf("failed to create a file in the directory: %w", err)
	}

	if err := probe.Close(); err != nil {
		return fmt.Errorf("failed to close the probe file: %w", err)
	}

	if err := os.Remove(probe.Name()); err != nil {
		return fmt.Errorf("failed to remove the probe file: %w", err)
	}

	return nil
}
