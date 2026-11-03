package display

import (
	"bufio"
	"fmt"
	"io"
	"net/netip"
)

func Prefixes(out io.Writer, prefixes []netip.Prefix) error {
	writer := bufio.NewWriter(out)
	for _, p := range prefixes {
		if _, err := fmt.Fprintln(writer, p); err != nil {
			return fmt.Errorf("failed to write the prefixes: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to write the prefixes: %w", err)
	}
	return nil
}
