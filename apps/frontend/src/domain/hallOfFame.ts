import {EndedSeason} from "../backends/hallOfFame.ts"
import {Score} from "../backends/standings.ts"

export const HALL_TABLE_ROWS = 10

export function championsOf(season: EndedSeason): Score[] {
    return season.scores.filter((score) => score.rank === 1)
}

export function finalTableOf(season: EndedSeason, countryCode: string): {listed: Score[], own?: Score} {
    const listed = season.scores.slice(0, HALL_TABLE_ROWS)
    if (listed.some((score) => score.countryCode === countryCode)) return {listed}
    return {listed, own: season.scores.find((score) => score.countryCode === countryCode)}
}

export function holds(hall: readonly EndedSeason[], season: number): boolean {
    return hall.some((ended) => ended.number === season)
}
