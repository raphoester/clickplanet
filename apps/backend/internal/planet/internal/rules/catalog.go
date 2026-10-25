package rules

import (
	"fmt"
	"slices"
)

func NewCatalog(config Config, defaults Set) (Catalog, error) {
	sets := map[SetName]Set{Default: {Name: Default, Tempo: defaults.Tempo, Boxes: defaults.Boxes}}

	for name, configured := range config.Sets {
		set := Set{Name: name, Tempo: defaults.Tempo, Boxes: defaults.Boxes}
		if configured.Tempo > 0 {
			set.Tempo = configured.Tempo
		}
		if configured.Boxes.MinInterval > 0 {
			set.Boxes.MinInterval = configured.Boxes.MinInterval
		}
		if configured.Boxes.MaxInterval > 0 {
			set.Boxes.MaxInterval = configured.Boxes.MaxInterval
		}
		if configured.Boxes.MaxChargesPerHour > 0 {
			set.Boxes.MaxChargesPerHour = configured.Boxes.MaxChargesPerHour
		}
		if set.Boxes.MinInterval > set.Boxes.MaxInterval {
			return Catalog{}, fmt.Errorf("rules.sets.%s: its boxes wait %s at least and %s at most", name,
				set.Boxes.MinInterval, set.Boxes.MaxInterval)
		}

		sets[name] = set
	}

	return Catalog{sets: sets}, nil
}

type Catalog struct {
	sets map[SetName]Set
}

func (c Catalog) Set(name SetName) (Set, error) {
	set, ok := c.sets[name]
	if !ok {
		return Set{}, fmt.Errorf("%q: %w", name, ErrUnknownSet)
	}

	return set, nil
}

func (c Catalog) SetOf(change Change) (Set, error) {
	if change == (Change{}) {
		return c.sets[Default], nil
	}

	return c.Set(change.set)
}

func (c Catalog) Sets() []Set {
	sets := make([]Set, 0, len(c.sets))
	for _, set := range c.sets {
		sets = append(sets, set)
	}

	slices.SortFunc(sets, func(a, b Set) int {
		switch {
		case a.Name == Default:
			return -1
		case b.Name == Default:
			return 1
		case a.Name < b.Name:
			return -1
		default:
			return 1
		}
	})

	return sets
}
