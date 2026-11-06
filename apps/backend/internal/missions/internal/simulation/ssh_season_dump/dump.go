package ssh_season_dump

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/simulation/csv_season"
)

const OpenSSH = "ssh"

// The SQL comes on the standard input, so no shell on the way reads it.
const psql = "docker exec -i cp-postgres psql -U clickplanet -d clickplanet -X -q -v ON_ERROR_STOP=1"

// Every day told that ends after the start.
const daysQuery = `COPY (
	SELECT day_id, season, number, (extract(epoch FROM starts_at) * 1000000)::bigint,
	       (extract(epoch FROM ends_at) * 1000000)::bigint
	FROM missions.mission_days WHERE ends_at > '%[1]s'::timestamptz ORDER BY starts_at
) TO STDOUT WITH (FORMAT csv)`

// Every act from the start, unfolded as planet's ledger kinds keep them (internal/planet/internal/ledger). The ledger
// has no index on time: a bisection over positions finds the first event at the start in some 40 index reads, so the
// read is a range of the primary key, never a scan of every season.
const actsQuery = `COPY (
	WITH RECURSIVE search(low, high) AS (
		SELECT min(position), max(position) + 1 FROM planet.ledger_events
		UNION ALL
		SELECT CASE WHEN probe.taken_at < '%[1]s'::timestamptz THEN probe.position + 1 ELSE search.low END,
		       CASE WHEN probe.taken_at < '%[1]s'::timestamptz THEN search.high ELSE (search.low + search.high) / 2 END
		FROM search
		LEFT JOIN LATERAL (
			SELECT position, taken_at FROM planet.ledger_events
			WHERE position >= (search.low + search.high) / 2 ORDER BY position LIMIT 1
		) AS probe ON true
		WHERE search.low < search.high
	)
	SELECT e.position, a.idx, (extract(epoch FROM e.taken_at) * 1000000)::bigint, a.act, a.flag, a.other, a.tile,
	       CASE WHEN e.account IS NULL THEN 'f' ELSE 't' END
	FROM planet.ledger_events AS e
	CROSS JOIN LATERAL (
		SELECT 0 AS idx, 'take' AS act, e.country AS flag, COALESCE(e.previous, '') AS other, e.tile::bigint AS tile
		WHERE e.kind = 'take'
		UNION ALL
		SELECT 0, 'strike', e.country, COALESCE(e.previous, ''), e.tile::bigint
		WHERE e.kind = 'strike'
		UNION ALL
		SELECT (n - 1)::int, 'take', e.country, COALESCE(taken->>'owner', ''), (taken->>'tile')::bigint
		FROM jsonb_array_elements(COALESCE(e.payload->'taken', '[]'::jsonb)) WITH ORDINALITY AS t(taken, n)
		WHERE e.kind IN ('spread', 'enclose')
		UNION ALL
		SELECT (n - 1)::int, 'strike', e.country, COALESCE(struck->>'owner', ''), (struck->>'tile')::bigint
		FROM jsonb_array_elements(COALESCE(e.payload->'struck', '[]'::jsonb)) WITH ORDINALITY AS s(struck, n)
		WHERE e.kind IN ('spread', 'enclose')
		UNION ALL
		SELECT (n - 1)::int, 'clear', '', COALESCE(cleared->>'owner', ''), (cleared->>'tile')::bigint
		FROM jsonb_array_elements(COALESCE(e.payload->'cleared', '[]'::jsonb)) WITH ORDINALITY AS c(cleared, n)
		WHERE e.kind = 'bomb'
		UNION ALL
		SELECT (n - 1)::int, 'strike', COALESCE(e.payload->>'flag', ''), COALESCE(struck->>'owner', ''),
		       (struck->>'tile')::bigint
		FROM jsonb_array_elements(COALESCE(e.payload->'struck', '[]'::jsonb)) WITH ORDINALITY AS b(struck, n)
		WHERE e.kind = 'bomb'
	) AS a
	WHERE e.position >= (SELECT low FROM search WHERE low >= high) AND e.taken_at >= '%[1]s'::timestamptz
) TO STDOUT WITH (FORMAT csv)`

// host: user@host of the machine postgres runs on. client: the ssh program.
func New(host, client string) (*Dump, error) {
	if host == "" {
		return nil, errors.New("the ssh host is empty")
	}
	return &Dump{host: host, client: client}, nil
}

type Dump struct {
	host   string
	client string
}

// Writes the days and the acts from since, a season's start, into the folder, as csv_season reads them.
func (d *Dump) Write(ctx context.Context, since time.Time, folder string) error {
	if err := os.MkdirAll(folder, 0o750); err != nil {
		return fmt.Errorf("failed to make %s: %w", folder, err)
	}
	at := since.UTC().Format(time.RFC3339Nano)
	if err := d.copy(ctx, fmt.Sprintf(daysQuery, at), filepath.Join(folder, csv_season.DaysFile)); err != nil {
		return fmt.Errorf("failed to copy the days: %w", err)
	}
	if err := d.copy(ctx, fmt.Sprintf(actsQuery, at), filepath.Join(folder, csv_season.ActsFile)); err != nil {
		return fmt.Errorf("failed to copy the acts: %w", err)
	}
	return nil
}

func (d *Dump) copy(ctx context.Context, query, path string) error {
	file, err := os.Create(path) //nolint:gosec // the operator names the folder.
	if err != nil {
		return fmt.Errorf("failed to make %s: %w", path, err)
	}
	var complained bytes.Buffer
	// "--" ends the options: a host that starts with a dash is a host, never an option of ssh.
	call := exec.CommandContext(ctx, d.client, "--", d.host, psql) //nolint:gosec // the operator names the host.
	call.Stdin = strings.NewReader(query)
	call.Stdout = file
	call.Stderr = &complained
	err = call.Run()
	if closed := file.Close(); err == nil && closed != nil {
		return fmt.Errorf("failed to write %s: %w", path, closed)
	}
	if err != nil {
		return fmt.Errorf("ssh failed: %w: %s", err, strings.TrimSpace(complained.String()))
	}
	return nil
}
