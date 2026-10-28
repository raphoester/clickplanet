import {ranked, TileChange} from "./changes.ts"

// flags: back out of the tiles halfway, so the painted flags change on screen. dive: back out at the end.
export type Look = "flags" | "dive"

export const LOOKS: readonly Look[] = ["flags", "dive"]

const MIN_LANDMASS_TILES = 20

// A landmass taken and taken back changed hands too: its holder is read this many times along the changes.
const CHECKPOINTS = 8

// A landmass this big reads as a painted flag at a continent's framing: Latvia, not Belgium.
export const READABLE_FLIP_TILES = 100

// holders: who held most of the landmass, in turn, from the first. tile: one tile of it, to say where it is.
export type Flip = {landmass: number, tiles: number, holders: string[], tile: number}

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
    const tiles = new Map<number, number>()
    const hold = (landmass: number, owner: string, count: number) => {
        let owners = held.get(landmass)
        if (!owners) held.set(landmass, owners = new Map())
        owners.set(owner, (owners.get(owner) ?? 0) + count)
    }
    for (let i = 0; i < landmassOf.length; i++) {
        const landmass = landmassOf[i]
        if (!landmasses.has(landmass)) continue
        sizes.set(landmass, (sizes.get(landmass) ?? 0) + 1)
        if (!tiles.has(landmass)) tiles.set(landmass, i + 1)
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
        .map(([landmass, seen]) => ({landmass, tiles: sizes.get(landmass) ?? 0, holders: seen, tile: tiles.get(landmass) ?? 0}))
        .sort((a, b) => b.tiles - a.tiles)
}

// The landmasses whose painted flag became the flag's, big enough to read from far: another's when the window opens,
// its at the end.
export function flippedTo(flips: readonly Flip[], flag: string): Flip[] {
    return flips.filter(({holders, tiles}) => tiles >= READABLE_FLIP_TILES && holders[0] !== flag && holders[holders.length - 1] === flag)
}
