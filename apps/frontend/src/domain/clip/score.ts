import {tally, TileChange} from "./changes.ts"

const GROUND_SHARE = 0.1

const MOST_GROUNDS = 4

// Tiles taken from another flag, a short window over a long one, and a war over several countries over one.
export function scoreOf(changes: readonly TileChange[], groundOf: (tile: number) => string | undefined, hours: number): number {
    const captured = changes.filter(({from}) => from !== undefined)
    if (captured.length === 0) return 0

    const grounds = [...tally(captured.flatMap(({tile}) => groundOf(tile) ?? [])).values()]
        .filter((count) => count >= captured.length * GROUND_SHARE).length

    return captured.length / Math.sqrt(Math.max(1, hours)) * Math.min(MOST_GROUNDS, Math.max(1, grounds))
}
