export type TileChange = {
    tile: number
    from: string | undefined
    to: string | undefined
    at: number
}

export function ownersAfter(opening: ReadonlyMap<number, string>, changes: readonly TileChange[]): Map<number, string> {
    const owners = new Map(opening)
    for (const {tile, to} of changes) {
        if (to === undefined) owners.delete(tile)
        else owners.set(tile, to)
    }
    return owners
}

export function tally<T>(values: Iterable<T>): Map<T, number> {
    const counts = new Map<T, number>()
    for (const value of values) counts.set(value, (counts.get(value) ?? 0) + 1)
    return counts
}

// Most first, and by code on a tie, so a clip comes out the same twice.
export function ranked(counts: ReadonlyMap<string, number>): [string, number][] {
    return [...counts].sort(([a, x], [b, y]) => y - x || a.localeCompare(b))
}
