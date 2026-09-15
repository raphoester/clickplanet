// Command rollout seeds the pre-postgres data files, drives live traffic through a deploy, and checks
// that nothing was lost. run.sh is the sequence around it.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: rollout seed|check-seed|traffic|dump|compare [flags]")
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "seed":
		err = seedCommand(os.Args[2:])
	case "check-seed":
		err = checkSeedCommand(os.Args[2:])
	case "traffic":
		err = trafficCommand(os.Args[2:])
	case "dump":
		err = dumpCommand(os.Args[2:])
	case "compare":
		err = compareCommand(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "rollout: %v\n", err)
		os.Exit(1)
	}
}

func seedCommand(args []string) error {
	flags := flag.NewFlagSet("seed", flag.ExitOnError)
	dir := flags.String("dir", "", "where to write the data files")
	maxIndex := flags.Uint("max-index", 257948, "gameMap.maxIndex")
	owned := flags.Float64("owned", 0.7, "share of tiles owned in the seed")
	_ = flags.Parse(args)

	if *dir == "" {
		return errors.New("-dir is required")
	}

	return writeSeed(*dir, uint32(*maxIndex), *owned) //nolint:gosec // a tile count.
}

func trafficCommand(args []string) error {
	flags := flag.NewFlagSet("traffic", flag.ExitOnError)
	addr := flags.String("addr", "http://127.0.0.1:18080", "the backend, bypassing Caddy")
	seedPath := flags.String("seed", "", "seed.json written by the seed command")
	marksPath := flags.String("marks", "", "file where run.sh writes deploy_start (the deploy command starts) and deploy_end (the backend answers again), unix ms")
	maxIndex := flags.Uint("max-index", 257948, "gameMap.maxIndex")
	players := flags.Int("players", 40, "concurrent players, one address each")
	clickEvery := flags.Duration("click-every", time.Second, "mean pause between one player's clicks")
	maxDowntime := flags.Duration("max-downtime", 30*time.Second, "longest outage the rollout may cause")
	settle := flags.Duration("settle", 3*time.Second, "how long after the deploy ends a failure is still forgiven")
	_ = flags.Parse(args)

	s, err := readSeed(*seedPath)
	if err != nil {
		return err
	}

	a := newAPI(*addr)
	perPlayer := (uint32(*maxIndex) + 1) / uint32(*players) //nolint:gosec // counts.
	t := newTraffic(a, *players, perPlayer, *clickEvery)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	fmt.Printf("traffic: %d players, a click every ~%s each, until interrupted\n", *players, *clickEvery)
	t.run(ctx)
	stop()

	return verify(a, t, s, *maxIndex, *marksPath, *maxDowntime, *settle)
}

func readSeed(path string) (seed, error) {
	var s seed
	raw, err := os.ReadFile(path) //nolint:gosec // a path the operator passes.
	if err != nil {
		return s, fmt.Errorf("read seed: %w", err)
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("decode seed: %w", err)
	}
	return s, nil
}

// verify prints the report and fails on lost data, a long outage, or errors once the deploy is over.
func verify(a *api, t *traffic, s seed, maxIndex uint, marksPath string, maxDowntime, settle time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var failures []string

	actual, err := a.readMap(ctx, 0, uint32(maxIndex)) //nolint:gosec // a tile count.
	if err != nil {
		return fmt.Errorf("read the final map: %w", err)
	}

	histories := t.histories()
	lostAcks, lostSeed, landedUnanswered := 0, 0, 0
	var examples []string
	for tile, owner := range actual {
		h, clicked := histories[tile]
		ok, byUnanswered := h.allows(owner, s.Owners[tile])
		switch {
		case ok && byUnanswered:
			landedUnanswered++
		case ok:
		case clicked && h.lastAck != nil:
			lostAcks++
		default:
			lostSeed++
		}
		if !ok && len(examples) < 5 {
			examples = append(examples, fmt.Sprintf("tile %d holds %q, seed %q, history %+v", tile, owner, s.Owners[tile], h))
		}
	}

	history, err := a.chatHistory(ctx)
	if err != nil {
		failures = append(failures, fmt.Sprintf("chat history unreadable: %v", err))
	}
	missingChat := 0
	for _, id := range s.ChatIDs {
		if !slices.Contains(history, id) {
			missingChat++
		}
	}

	acks := t.acked()
	t.mu.Lock()
	clicks := outageOf(t.clicks)
	probes := outageOf(t.probes)
	refused := fmt.Sprint(t.refused)
	after := failuresAfter(t.clicks, marksPath, settle) + failuresAfter(t.probes, marksPath, settle)
	t.mu.Unlock()

	fmt.Println()
	fmt.Println("=== rollout report ===")
	fmt.Printf("clicks: %d sent, %d OK, %d failed; refusals by HTTP status (0 = no answer): %s\n",
		clicks.total, acks, clicks.failed, refused)
	fmt.Printf("probes: %d sent, %d failed\n", probes.total, probes.failed)
	fmt.Printf("downtime (longest gap between two OK probes): %s, from %s to %s\n",
		probes.longest.Round(time.Millisecond), clock(probes.from), clock(probes.to))
	fmt.Printf("longest gap between two OK clicks: %s\n", clicks.longest.Round(time.Millisecond))
	fmt.Printf("tiles clicked: %d; their last OK click lost: %d\n", len(histories), lostAcks)
	fmt.Printf("clicks with no answer that still landed (not a loss): %d\n", landedUnanswered)
	fmt.Printf("seeded tiles lost or changed by nothing the traffic sent: %d (of %d seeded)\n", lostSeed, s.OwnedTiles)
	for _, example := range examples {
		fmt.Println("  " + example)
	}
	fmt.Printf("seeded chat messages missing: %d of %d\n", missingChat, len(s.ChatIDs))
	fmt.Printf("failures more than %s after the deploy ended: %d\n", settle, after)
	fmt.Println(deployLine(marksPath))

	if lostAcks > 0 {
		failures = append(failures, fmt.Sprintf("%d acknowledged clicks lost", lostAcks))
	}
	if lostSeed > 0 {
		failures = append(failures, fmt.Sprintf("%d seeded tiles lost or changed", lostSeed))
	}
	if missingChat > 0 {
		failures = append(failures, fmt.Sprintf("%d seeded chat messages missing", missingChat))
	}
	if probes.longest > maxDowntime {
		failures = append(failures, fmt.Sprintf("downtime %s is over %s", probes.longest, maxDowntime))
	}
	if after > 0 {
		failures = append(failures, fmt.Sprintf("%d requests failed after the deploy settled", after))
	}
	if acks == 0 {
		failures = append(failures, "no click was ever accepted: the traffic never reached the game")
	}

	if len(failures) > 0 {
		return fmt.Errorf("FAIL: %s", strings.Join(failures, "; "))
	}

	fmt.Println("PASS")
	return nil
}

func failuresAfter(samples []sample, marksPath string, settle time.Duration) int {
	_, end, ok := readMarks(marksPath)
	if !ok {
		return 0
	}

	failed := 0
	for _, s := range samples {
		if !s.ok && s.at.After(end.Add(settle)) {
			failed++
		}
	}
	return failed
}

func deployLine(marksPath string) string {
	start, end, ok := readMarks(marksPath)
	if !ok {
		return "deploy: no marks recorded"
	}
	return fmt.Sprintf("deploy, from the command to serving again: %s, from %s to %s", end.Sub(start).Round(time.Millisecond), clock(start), clock(end))
}

func readMarks(path string) (time.Time, time.Time, bool) {
	raw, err := os.ReadFile(path) //nolint:gosec // a path the operator passes.
	if err != nil {
		return time.Time{}, time.Time{}, false
	}

	marks := map[string]time.Time{}
	for line := range strings.Lines(string(raw)) {
		name, value, found := strings.Cut(strings.TrimSpace(line), " ")
		if !found {
			continue
		}
		ms, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			continue
		}
		marks[name] = time.UnixMilli(ms)
	}

	start, hasStart := marks["deploy_start"]
	end, hasEnd := marks["deploy_end"]
	return start, end, hasStart && hasEnd
}

func clock(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("15:04:05.000")
}
