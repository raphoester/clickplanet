import {Season} from "../backends/season.ts"
import {ClosedRound} from "../backends/standings.ts"

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
