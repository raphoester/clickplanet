import type {MySeason, Standing} from "../backends/standings.ts"

export const STANDINGS_SHOWN = 10

export type Takes = ReadonlyMap<string, number>

export const NO_TAKES: Takes = new Map()

export function withTake(takes: Takes, country: string): Takes {
    return new Map(takes).set(country, (takes.get(country) ?? 0) + 1)
}

export function takesSince(before: Takes, now: Takes): Takes {
    const since = new Map<string, number>()
    for (const [country, count] of now) {
        const more = count - (before.get(country) ?? 0)
        if (more > 0) since.set(country, more)
    }
    return since
}

export function takenCount(takes: Takes): number {
    let count = 0
    for (const taken of takes.values()) count += taken
    return count
}

export function liveSeason(read: MySeason, since: Takes): MySeason {
    const main = read.countryCode ?? mainOf(since)
    const more = main === undefined ? 0 : since.get(main) ?? 0
    return more === 0 ? read : {...read, countryCode: main, tiles: read.tiles + more}
}

// Mirrors the server's Tally.WithTake: change both together.
function mainOf(takes: Takes): string | undefined {
    let main: string | undefined
    for (const [country, count] of takes) {
        if (main === undefined || count > (takes.get(main) ?? 0)) main = country
    }
    return main
}

export type Board = {
    listed: Standing[]
    below?: Standing
}

export function boardWith(standings: readonly Standing[], own: Standing | undefined): Board {
    if (!own) return {listed: [...standings]}

    const others = standings.filter((standing) => standing.name !== own.name)
    const at = others.findIndex((standing) => standing.tiles < own.tiles)
    const merged = at === -1 ? [...others, own] : [...others.slice(0, at), own, ...others.slice(at)]
    const listed = ranked(merged).slice(0, STANDINGS_SHOWN)

    return listed.some((standing) => standing.name === own.name) ? {listed} : {listed, below: own}
}

function ranked(standings: Standing[]): Standing[] {
    const out: Standing[] = []
    for (const [index, standing] of standings.entries()) {
        const previous = out[index - 1]
        const rank = previous?.tiles === standing.tiles ? previous.rank : index + 1
        out.push(rank === standing.rank ? standing : {...standing, rank})
    }
    return out
}
