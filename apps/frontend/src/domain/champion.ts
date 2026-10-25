import {Season} from "../backends/season.ts"
import {ClosedRound} from "../backends/standings.ts"
import {Countries} from "./countries.ts"

export type Champion = {
    season: number
    countryCodes: string[]
}

export function championOf(closed: ClosedRound | undefined): Champion | undefined {
    if (!closed?.finale) return undefined
    const countryCodes = closed.after.filter((score) => score.rank === 1).map((score) => score.countryCode)
    return countryCodes.length > 0 ? {season: closed.season, countryCodes} : undefined
}

export function reigningChampionOf(season: Season | undefined, closed: ClosedRound | undefined): Champion | undefined {
    const champion = championOf(closed)
    if (!champion || (season && season.number > champion.season)) return undefined
    return champion
}

export function championNames(champion: Champion): string {
    const names = champion.countryCodes.map((code) => Countries.get(code)?.name ?? code)
    return names.length > 1 ? `${names.slice(0, -1).join(", ")} and ${names[names.length - 1]}` : names[0]
}

export function winsLine(champion: Champion): string {
    return `${championNames(champion)} ${champion.countryCodes.length > 1 ? "win" : "wins"} Season ${champion.season}`
}
