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

// A country fought over this much is a story of its own, told on its own ground, besides the wars across the map.
const FOUGHT_OVER_LEAST = 400

const MOST_COUNTRIES = 12

// The countries whose ground was taken from one flag by another the most between two times, the most first.
export function foughtOver(
    changes: readonly TileChange[],
    groundOf: (tile: number) => string | undefined,
    since: number,
    until: number,
): string[] {
    const grounds = tally(changes.flatMap(({tile, from, to, at}) => {
        const ground = groundOf(tile)
        return at < since || at > until || from === undefined || to === undefined || ground === undefined ? [] : [ground]
    }))
    return ranked(grounds).filter(([, count]) => count >= FOUGHT_OVER_LEAST).slice(0, MOST_COUNTRIES).map(([country]) => country)
}

// The flags that lost the most tiles to another flag, the most first.
export function losersOf(changes: readonly TileChange[]): string[] {
    return ranked(tally(changes.flatMap(({from, to}) => from === undefined || to === undefined || from === to ? [] : [from])))
        .map(([flag]) => flag)
}

// The flags that took the most from one flag, the most first.
export function takersFrom(changes: readonly TileChange[], loser: string): string[] {
    return ranked(tally(changes.flatMap(({from, to}) => from === loser && to !== undefined && to !== loser ? [to] : [])))
        .map(([flag]) => flag)
}
