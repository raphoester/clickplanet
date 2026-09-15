package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

// checkSeedCommand fails unless the running backend holds exactly the seeded map.
func checkSeedCommand(args []string) error {
	flags := flag.NewFlagSet("check-seed", flag.ExitOnError)
	addr := flags.String("addr", "http://127.0.0.1:18080", "the backend")
	seedPath := flags.String("seed", "", "seed.json written by the seed command")
	maxIndex := flags.Uint("max-index", 257948, "gameMap.maxIndex")
	_ = flags.Parse(args)

	s, err := readSeed(*seedPath)
	if err != nil {
		return err
	}

	actual, err := readWholeMap(newAPI(*addr), *maxIndex)
	if err != nil {
		return err
	}

	differ := 0
	for tile, owner := range actual {
		if s.Owners[tile] != owner {
			differ++
		}
	}
	if differ > 0 {
		return fmt.Errorf("the backend does not hold the seeded map: %d tiles differ", differ)
	}

	fmt.Printf("the backend holds the seeded map: %d owned tiles\n", s.OwnedTiles)
	return nil
}

// dumpCommand writes the backend's map to a file, and prints how many tiles are owned.
func dumpCommand(args []string) error {
	flags := flag.NewFlagSet("dump", flag.ExitOnError)
	addr := flags.String("addr", "http://127.0.0.1:18080", "the backend")
	out := flags.String("out", "", "where to write the map")
	maxIndex := flags.Uint("max-index", 257948, "gameMap.maxIndex")
	_ = flags.Parse(args)

	if *out == "" {
		return errors.New("-out is required")
	}

	actual, err := readWholeMap(newAPI(*addr), *maxIndex)
	if err != nil {
		return err
	}

	raw, err := json.Marshal(actual)
	if err != nil {
		return fmt.Errorf("encode the map: %w", err)
	}

	fmt.Println(owned(actual))
	return os.WriteFile(*out, raw, 0o600)
}

// compareCommand fails unless the backend's map is the one a dump recorded.
func compareCommand(args []string) error {
	flags := flag.NewFlagSet("compare", flag.ExitOnError)
	addr := flags.String("addr", "http://127.0.0.1:18080", "the backend")
	with := flags.String("with", "", "a map written by dump")
	maxIndex := flags.Uint("max-index", 257948, "gameMap.maxIndex")
	_ = flags.Parse(args)

	raw, err := os.ReadFile(*with) //nolint:gosec // a path the operator passes.
	if err != nil {
		return fmt.Errorf("read the dump: %w", err)
	}
	var want map[uint32]string
	if err := json.Unmarshal(raw, &want); err != nil {
		return fmt.Errorf("decode the dump: %w", err)
	}

	actual, err := readWholeMap(newAPI(*addr), *maxIndex)
	if err != nil {
		return err
	}

	differ := 0
	for tile, owner := range want {
		if actual[tile] != owner {
			differ++
		}
	}
	if differ > 0 {
		return fmt.Errorf("the map changed across the restart: %d tiles differ", differ)
	}

	fmt.Printf("the map is the same after the restart: %d owned tiles\n", owned(actual))
	return nil
}

func readWholeMap(a *api, maxIndex uint) (map[uint32]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	actual, err := a.readMap(ctx, 0, uint32(maxIndex)) //nolint:gosec // a tile count.
	if err != nil {
		return nil, fmt.Errorf("read the map: %w", err)
	}

	return actual, nil
}

func owned(owners map[uint32]string) int {
	count := 0
	for _, owner := range owners {
		if owner != "" {
			count++
		}
	}
	return count
}
