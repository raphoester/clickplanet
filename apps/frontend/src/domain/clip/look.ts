import {ranked, TileChange} from "./changes.ts"

// flags: back out of the tiles halfway, so the painted flags change on screen. dive: back out at the end.
export type Look = "flags" | "dive"

// Below TILES_ZOOM a tile is a few pixels and its flag a blur: tiles only move on screen above it.
export const TILES_ZOOM = 3

const MIN_LANDMASS_TILES = 20

const MIN_FLIPS = 2

// About two Frances: less than that changing its painted flag is a still picture from far.
const FLAGS_GROUND = 2000

// A landmass taken and taken back changed hands too: its holder is read this many times along the changes.
const CHECKPOINTS = 8

// holders: who held most of the landmass, in turn, from the first.
export type Flip = {tiles: number, holders: string[]}

// A landmass flips when the flag holding most of it changes: the only change its painted flag shows from far.
// landmassOf holds each tile's landmass, tile id - 1 first, 0 for none. changes are in the order they happened.
export function flipsOf(
    landmassOf: ArrayLike<number>,
    opening: ReadonlyMap<number, string>,
    changes: readonly TileChange[],
    touched: readonly number[],
): Flip[] {
    const landmasses = new Set(touched.map((tile) => landmassOf[tile - 1]).filter((landmass) => landmass > 0))
    const held = new Map<number, Map<string, number>>()
    const sizes = new Map<number, number>()
    const hold = (landmass: number, owner: string, count: number) => {
        let owners = held.get(landmass)
        if (!owners) held.set(landmass, owners = new Map())
        owners.set(owner, (owners.get(owner) ?? 0) + count)
    }
    for (let i = 0; i < landmassOf.length; i++) {
        const landmass = landmassOf[i]
        if (!landmasses.has(landmass)) continue
        sizes.set(landmass, (sizes.get(landmass) ?? 0) + 1)
        hold(landmass, opening.get(i + 1) ?? "", 1)
    }

    const holders = new Map([...held].map(([landmass, owners]) => [landmass, [ranked(owners)[0][0]]]))
    const read = () => {
        for (const [landmass, owners] of held) {
            const seen = holders.get(landmass)!
            const now = ranked(owners)[0][0]
            if (seen[seen.length - 1] !== now) seen.push(now)
        }
    }
    const step = Math.max(1, Math.ceil(changes.length / CHECKPOINTS))
    changes.forEach(({tile, from, to}, i) => {
        const landmass = landmassOf[tile - 1]
        if (landmasses.has(landmass)) {
            hold(landmass, from ?? "", -1)
            hold(landmass, to ?? "", 1)
        }
        if ((i + 1) % step === 0 || i === changes.length - 1) read()
    })

    return [...holders]
        .filter(([landmass, seen]) => seen.length > 1 && (sizes.get(landmass) ?? 0) >= MIN_LANDMASS_TILES)
        .map(([landmass, seen]) => ({tiles: sizes.get(landmass) ?? 0, holders: seen}))
        .sort((a, b) => b.tiles - a.tiles)
}

// A wide front where big land changed hands shows its painted flags change for half the clip; anything else dives.
export function lookOf(flips: readonly Flip[], zoom: number): Look {
    const ground = flips.reduce((sum, {tiles}) => sum + tiles, 0)
    return zoom < TILES_ZOOM && flips.length >= MIN_FLIPS && ground >= FLAGS_GROUND ? "flags" : "dive"
}
