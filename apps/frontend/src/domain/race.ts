import {Race} from "../backends/standings.ts"
import {Countries, Country} from "./countries.ts"
import {LeaderboardEntry} from "./leaderboard.ts"

export type CountryOrder = "season" | "territory"

export type CountryLine = {
    country: Country
    tiles: number
    points: number
    today: number
}

export function countryLines(
    leaderboard: readonly LeaderboardEntry[],
    race: Race | undefined,
    order: CountryOrder,
): CountryLine[] {
    const today = new Map(race?.round?.standings.map((standing) => [standing.countryCode, standing.points]))
    const points = new Map(race?.scores.map((score) => [score.countryCode, score.points]))
    const line = (entry: LeaderboardEntry): CountryLine => ({
        ...entry,
        points: points.get(entry.country.code) ?? 0,
        today: today.get(entry.country.code) ?? 0,
    })

    const lines = leaderboard.map(line)
    if (order === "territory") return lines

    const held = new Set(leaderboard.map((entry) => entry.country.code))
    for (const code of new Set([...points.keys(), ...today.keys()])) {
        const country = Countries.get(code)
        if (!held.has(code) && country) lines.push(line({country, tiles: 0}))
    }
    return lines
        .filter((l) => l.tiles > 0 || l.points + l.today > 0)
        .sort((a, b) => b.points - a.points || b.tiles - a.tiles || a.country.code.localeCompare(b.country.code))
}
