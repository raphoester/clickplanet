package display

import (
	"bufio"
	"fmt"
	"io"

	"github.com/raphoester/clickplanet.lol-backend/cmd/migrationcheck/internal/migrations"
)

func Problems(out io.Writer, problems []migrations.Problem) (bool, error) {
	writer := bufio.NewWriter(out)
	for _, p := range problems {
		if _, err := fmt.Fprintf(writer, "%s: %s\n", p.Directory, p.Reason); err != nil {
			return false, fmt.Errorf("failed to write the problems: %w", err)
		}
	}
	if len(problems) == 0 {
		if _, err := fmt.Fprintln(writer, "the migrations are in order"); err != nil {
			return false, fmt.Errorf("failed to write the problems: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return false, fmt.Errorf("failed to write the problems: %w", err)
	}
	return len(problems) == 0, nil
}
