// A throwaway driver for the local proof. Not committed.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_geodesic_map"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
)

func main() {
	m := embedded_geodesic_map.New(260503, slog.New(slog.DiscardHandler))
	geo, err := m.LoadGeography()
	if err != nil {
		panic(err)
	}
	borders, err := m.LoadBorders()
	if err != nil {
		panic(err)
	}
	caps, err := m.LoadCapitals(borders)
	if err != nil {
		panic(err)
	}
	caps = caps.Flags(cpcountries.New().CheckCountry)
	tileOf := map[string]uint32{}
	for _, c := range caps.All() {
		tileOf[c.Country] = c.Tile
	}
	switch os.Args[1] {
	case "capital":
		fmt.Println(tileOf[os.Args[2]], geo.Neighbours(tileOf[os.Args[2]]))
	case "path": // path from a's capital to b's capital, over land tiles
		from, to := tileOf[os.Args[2]], tileOf[os.Args[3]]
		prev := map[uint32]uint32{from: 0}
		queue := []uint32{from}
		for len(queue) > 0 {
			t := queue[0]
			queue = queue[1:]
			if t == to {
				break
			}
			for _, n := range geo.Neighbours(t) {
				if _, seen := prev[n]; !seen {
					prev[n] = t
					queue = append(queue, n)
				}
			}
		}
		var path []uint32
		for t := to; t != 0; t = prev[t] {
			path = append([]uint32{t}, path...)
		}
		fmt.Println(len(path), path)
	case "far":
		n, _ := strconv.Atoi(os.Args[3])
		fmt.Println(tileOf[os.Args[2]] + uint32(n))
	}
}
