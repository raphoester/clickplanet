import {Race} from "../backends/standings.ts"

export const RACE_SHOWN = 10

export type RaceRow = {
    rank: number
    countryCode: string
    points: number
    today: number
}

export type RaceTable = {
    listed: RaceRow[]
    below?: RaceRow
}

export function raceTable(race: Race, countryCode: string): RaceTable {
    const today = new Map(race.round?.standings.map((standing) => [standing.countryCode, standing]))
    const points = new Map(race.scores.map((score) => [score.countryCode, score.points]))
    const codes = new Set([...points.keys(), ...[...today.values()].filter((s) => s.points > 0).map((s) => s.countryCode)])

    const rows = [...codes]
        .map((code) => ({rank: 0, countryCode: code, points: points.get(code) ?? 0, today: today.get(code)?.points ?? 0}))
        .sort((a, b) =>
            total(b) - total(a)
            || (today.get(a.countryCode)?.rank ?? Infinity) - (today.get(b.countryCode)?.rank ?? Infinity)
            || a.countryCode.localeCompare(b.countryCode))
    rows.forEach((row, i) => row.rank = i > 0 && total(rows[i - 1]) === total(row) ? rows[i - 1].rank : i + 1)

    const listed = rows.slice(0, RACE_SHOWN)
    const below = rows.slice(RACE_SHOWN).find((row) => row.countryCode === countryCode)
    return below ? {listed, below} : {listed}
}

function total(row: RaceRow): number {
    return row.points + row.today
}
