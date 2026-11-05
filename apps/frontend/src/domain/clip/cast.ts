// A tile a player of the cast took from another flag, for flag.
export type Take = {at: number, tile: number, flag: string, fighter: number}

// Who took which tiles in a window, under numbers of one run's cast alone: each player's name by its number.
export type Cast = {id: string, names: ReadonlyMap<number, string>, takes: readonly Take[]}

export type TopFighter = {number: number, name: string, took: number}

// The players a clip names on one of its sides, under that side's flag.
export type NamedSide = {flag: string, fighters: TopFighter[]}

export const NO_CAST: Cast = {id: "", names: new Map(), takes: []}

const MOST_FIGHTERS = 3

// The cast's times are the database's and the replay's are the stream's: a take is matched to what the replay played
// by its tile and a stretch of time, never by its time.
export function castWithin(cast: Cast, since: number, until: number): Cast {
    return {...cast, takes: cast.takes.filter(({at}) => at >= since && at <= until)}
}

function ranked(cast: Cast, takes: Iterable<Take>): TopFighter[] {
    const took = new Map<number, number>()
    for (const {fighter} of takes) took.set(fighter, (took.get(fighter) ?? 0) + 1)
    return [...took]
        .sort(([a, x], [b, y]) => y - x || a - b)
        .flatMap(([number, count]) => {
            const name = cast.names.get(number)
            return name === undefined ? [] : [{number, name, took: count}]
        })
}

// takes: the ones counted, of the cast's.
export function topFightersOf(cast: Cast, takes: Iterable<Take>): TopFighter[] {
    return ranked(cast, takes).slice(0, MOST_FIGHTERS)
}

// The players the editor named, each under the side it took the most for in what the clip plays (takes), the sides in
// their order and the most first on each. sideOf: the side a flag fights on, when not its own. One who took nothing for
// a side there is not named.
export function namedSidesOf(
    cast: Cast,
    numbers: readonly number[],
    takes: readonly Take[],
    sides: readonly string[],
    sideOf: (flag: string) => string,
): NamedSide[] {
    for (const number of numbers) {
        if (!cast.names.has(number)) throw new Error(`no fighter number ${number} in the cast`)
    }
    const named = new Set(numbers)
    const bySide = sides.map((side) => ranked(cast, takes.filter(({flag, fighter}) => named.has(fighter) && sideOf(flag) === side)))
    const sideOfFighter = (number: number) => {
        const took = bySide.map((fighters) => fighters.find((fighter) => fighter.number === number)?.took ?? 0)
        return took.indexOf(Math.max(...took))
    }
    return sides.flatMap((flag, side) => {
        const fighters = bySide[side].filter(({number}) => sideOfFighter(number) === side).slice(0, MOST_FIGHTERS)
        return fighters.length === 0 ? [] : [{flag, fighters}]
    })
}
