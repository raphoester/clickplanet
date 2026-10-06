import {ranked, tally} from "./changes.ts"

// flags: painted flags all along. tiles: every tile all along. dive: painted flags from far, tiles up close between.
export type Look = "flags" | "tiles" | "dive"

// Below TILES_ZOOM a tile is a few pixels and its flag a blur: tiles only move on screen above it.
export const TILES_ZOOM = 3

const MIN_LANDMASS_TILES = 20

const MIN_FLIPS = 2

// About two Frances: less than that changing its painted flag is a still picture from far.
const FLAGS_GROUND = 2000

// A dive pulls back out to show the land that changed hands, so some has to be big enough to see from far.
const REVEAL_TILES = 300

export type Flip = {tiles: number, was: string, is: string}

// A landmass flips when the flag holding most of it changes: the only change its painted flag shows from far.
// landmassOf holds each tile's landmass, tile id - 1 first, 0 for none.
export function flipsOf(
    landmassOf: ArrayLike<number>,
    before: ReadonlyMap<number, string>,
    after: ReadonlyMap<number, string>,
    touched: readonly number[],
): Flip[] {
    const landmasses = new Set(touched.map((tile) => landmassOf[tile - 1]).filter((landmass) => landmass > 0))
    const owners = new Map<number, {was: string[], is: string[]}>()
    for (let i = 0; i < landmassOf.length; i++) {
        const landmass = landmassOf[i]
        if (!landmasses.has(landmass)) continue
        let held = owners.get(landmass)
        if (!held) owners.set(landmass, held = {was: [], is: []})
        held.was.push(before.get(i + 1) ?? "")
        held.is.push(after.get(i + 1) ?? "")
    }

    const flips: Flip[] = []
    for (const {was, is} of owners.values()) {
        if (was.length < MIN_LANDMASS_TILES) continue
        const flip = {tiles: was.length, was: ranked(tally(was))[0][0], is: ranked(tally(is))[0][0]}
        if (flip.was !== flip.is) flips.push(flip)
    }
    return flips.sort((a, b) => b.tiles - a.tiles)
}

// A wide front is painted flags when big land changed hands. Any front is a dive when some did, and tiles otherwise.
export function lookOf(flips: readonly Flip[], zoom: number): Look {
    const ground = flips.reduce((sum, {tiles}) => sum + tiles, 0)
    if (zoom < TILES_ZOOM && flips.length >= MIN_FLIPS && ground >= FLAGS_GROUND) return "flags"
    return flips.some(({tiles}) => tiles >= REVEAL_TILES) ? "dive" : "tiles"
}
