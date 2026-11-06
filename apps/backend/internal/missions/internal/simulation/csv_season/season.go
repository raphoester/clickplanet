package csv_season

import (
	"cmp"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/simulation"
)

// The files ssh_season_dump writes in a folder.
const (
	DaysFile = "days.csv"
	ActsFile = "acts.csv"
)

var ErrUnknownAct = errors.New("not an act the dump writes")

// Every day told, and every act in ledger order.
func Read(folder string) ([]days.Day, []simulation.Act, error) {
	calendar, err := readDays(filepath.Join(folder, DaysFile))
	if err != nil {
		return nil, nil, err
	}
	acts, err := readActs(filepath.Join(folder, ActsFile))
	if err != nil {
		return nil, nil, err
	}
	return calendar, acts, nil
}

// day_id, season, number, and starts_at and ends_at in microseconds since 1970.
func readDays(path string) ([]days.Day, error) {
	var calendar []days.Day
	err := eachRecord(path, 5, func(record []string) error {
		season, err := strconv.ParseUint(record[1], 10, 32)
		if err != nil {
			return fmt.Errorf("failed to read a season: %w", err)
		}
		number, err := strconv.ParseUint(record[2], 10, 32)
		if err != nil {
			return fmt.Errorf("failed to read a day's number: %w", err)
		}
		starts, err := microsOf(record[3])
		if err != nil {
			return err
		}
		ends, err := microsOf(record[4])
		if err != nil {
			return err
		}
		day, err := days.DayOf(days.DaySnapshot{
			ID: days.DayID(record[0]), Season: uint32(season), Number: uint32(number), StartsAt: starts, EndsAt: ends,
		})
		if err != nil {
			return fmt.Errorf("failed to read a day: %w", err)
		}
		calendar = append(calendar, day)
		return nil
	})
	return calendar, err
}

type act struct {
	simulation.Act

	position uint64
	kind     int
	index    uint32
}

// position, index, at in microseconds since 1970, take, strike or clear, flag, the flag before or the shield's owner,
// tile, and t when an account made it.
func readActs(path string) ([]simulation.Act, error) {
	var read []act
	err := eachRecord(path, 8, func(record []string) error {
		position, err := strconv.ParseUint(record[0], 10, 64)
		if err != nil {
			return fmt.Errorf("failed to read a position: %w", err)
		}
		index, err := strconv.ParseUint(record[1], 10, 32)
		if err != nil {
			return fmt.Errorf("failed to read an index: %w", err)
		}
		at, err := microsOf(record[2])
		if err != nil {
			return err
		}
		tile, err := strconv.ParseUint(record[6], 10, 32)
		if err != nil {
			return fmt.Errorf("failed to read a tile: %w", err)
		}
		flag, other := record[4], record[5]

		switch record[3] {
		case "take":
			read = append(read, act{position: position, kind: 0, index: uint32(index), Act: simulation.Take{
				Position: position, Index: uint32(index), At: at, Flag: flag, Previous: other, Tile: uint32(tile),
				Account: record[7] == "t",
			}})
		case "strike":
			read = append(read, act{position: position, kind: 1, index: uint32(index), Act: simulation.Strike{
				Position: position, Index: uint32(index), At: at, Flag: flag, Owner: other, Tile: uint32(tile),
			}})
		case "clear":
			read = append(read, act{position: position, kind: 2, index: uint32(index), Act: simulation.Clear{
				Position: position, Index: uint32(index), At: at, Previous: other, Tile: uint32(tile),
			}})
		default:
			return fmt.Errorf("%w: %q", ErrUnknownAct, record[3])
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(read, func(a, b act) int {
		return cmp.Or(cmp.Compare(a.position, b.position), cmp.Compare(a.kind, b.kind), cmp.Compare(a.index, b.index))
	})
	acts := make([]simulation.Act, 0, len(read))
	for _, each := range read {
		acts = append(acts, each.Act)
	}
	return acts, nil
}

func eachRecord(path string, fields int, see func([]string) error) error {
	file, err := os.Open(path) //nolint:gosec // the operator names the folder.
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = fields
	reader.ReuseRecord = true
	for line := 1; ; line++ {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		if err := see(record); err != nil {
			return fmt.Errorf("%s, line %d: %w", path, line, err)
		}
	}
}

func microsOf(field string) (time.Time, error) {
	micros, err := strconv.ParseInt(field, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read a time: %w", err)
	}
	return time.UnixMicro(micros).UTC(), nil
}
