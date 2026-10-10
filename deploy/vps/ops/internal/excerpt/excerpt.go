package excerpt

import (
	"bufio"
	"fmt"
	"io"
)

const longestLine = 1 << 20

type Limit struct {
	Lines int
	Bytes int
}

type Excerpt struct {
	Lines     []string
	Truncated bool
}

func NewCollector(limit Limit) *Collector {
	return &Collector{limit: limit}
}

type Collector struct {
	limit     Limit
	lines     []string
	bytes     int
	truncated bool
}

func (c *Collector) Add(line string) {
	if c.truncated {
		return
	}
	if len(c.lines) >= c.limit.Lines || c.bytes+len(line) > c.limit.Bytes {
		c.truncated = true
		return
	}
	c.lines = append(c.lines, line)
	c.bytes += len(line)
}

func (c *Collector) Fill(source io.Reader, keep func(line []byte) bool) error {
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 64<<10), longestLine)
	for !c.truncated && scanner.Scan() {
		if keep(scanner.Bytes()) {
			c.Add(scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read a line: %w", err)
	}
	return nil
}

func (c *Collector) Full() bool {
	return c.truncated
}

func (c *Collector) Excerpt() Excerpt {
	return Excerpt{Lines: c.lines, Truncated: c.truncated}
}
