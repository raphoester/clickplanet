package journal

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/raphoester/clickplanet.lol-ops/internal/excerpt"
)

func NewJournalctl(directory string) *Journalctl {
	return &Journalctl{directory: directory}
}

type Journalctl struct {
	directory string
}

func (j *Journalctl) Read(ctx context.Context, query Query) (excerpt.Excerpt, error) {
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	command := exec.CommandContext(ctx, "journalctl", arguments(j.directory, query)...)
	var complaint bytes.Buffer
	command.Stderr = &complaint
	output, err := command.StdoutPipe()
	if err != nil {
		return excerpt.Excerpt{}, fmt.Errorf("failed to pipe journalctl: %w", err)
	}
	if err := command.Start(); err != nil {
		return excerpt.Excerpt{}, fmt.Errorf("failed to start journalctl: %w", err)
	}

	wanted := []byte(query.Contains)
	collector := excerpt.NewCollector(query.Limit)
	fillErr := collector.Fill(output, func(line []byte) bool { return bytes.Contains(line, wanted) })

	// Nobody reads the rest; stop journalctl rather than wait for it to block on a full pipe.
	if collector.Full() || fillErr != nil {
		stop()
	}
	waitErr := command.Wait()

	switch {
	case fillErr != nil:
		return excerpt.Excerpt{}, fmt.Errorf("failed to read journalctl: %w", fillErr)
	case waitErr != nil && !collector.Full():
		return excerpt.Excerpt{}, fmt.Errorf("journalctl failed: %w: %s", waitErr, bytes.TrimSpace(complaint.Bytes()))
	}
	return collector.Excerpt(), nil
}

func arguments(directory string, query Query) []string {
	return []string{
		"--directory", directory,
		"--quiet",
		"--no-pager",
		"--output", "cat",
		"--since", epoch(query.Since),
		"--until", epoch(query.Until),
		"CONTAINER_TAG=" + string(query.Tag),
	}
}

func epoch(moment time.Time) string {
	return "@" + strconv.FormatInt(moment.Unix(), 10)
}
