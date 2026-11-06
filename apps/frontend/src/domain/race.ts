import {Race, Round, Score} from "../backends/standings.ts"
import {Countries, Country} from "./countries.ts"
import {LeaderboardEntry} from "./leaderboard.ts"

export type CountryOrder = "points" | "tiles" | "gain"

export type CountryLine = {
    country: Country
    tiles: number
    points: number
    today: number
    // The tiles gained since the round that scores now opened; none while no round runs.
    gain: number | undefined
}

export function shownOrder(race: Race | undefined, picked: CountryOrder = "points"): CountryOrder {
    if (!race) return "tiles"
    if (picked === "gain" && !scoringNow(race)) return "points"
    return picked
}

export function countryLines(
    leaderboard: readonly LeaderboardEntry[],
    race: Race | undefined,
    order: CountryOrder,
): CountryLine[] {
    const now = scoringNow(race)
    const today = new Map(now?.standings.map((standing) => [standing.countryCode, standing.points]))
    const start = new Map(now?.start.map((held) => [held.countryCode, held.tiles]))
    const points = new Map(race?.scores.map((score) => [score.countryCode, score.points]))
    const line = (entry: LeaderboardEntry): CountryLine => ({
        ...entry,
        points: points.get(entry.country.code) ?? 0,
        today: today.get(entry.country.code) ?? 0,
        gain: now ? entry.tiles - (start.get(entry.country.code) ?? 0) : undefined,
    })

    const lines = leaderboard.map(line)
    if (order === "tiles") return lines

    const held = new Set(leaderboard.map((entry) => entry.country.code))
    for (const code of new Set([...points.keys(), ...today.keys()])) {
        const country = Countries.get(code)
        if (!held.has(code) && country) lines.push(line({country, tiles: 0}))
    }
    return lines
        .filter((l) => l.tiles > 0 || l.points + l.today > 0)
        .sort((a, b) => lead(b, order) - lead(a, order) || b.tiles - a.tiles || a.country.code.localeCompare(b.country.code))
}

function lead(line: CountryLine, order: "points" | "gain"): number {
    switch (order) {
        case "points":
            return line.points
        case "gain":
            return line.gain ?? 0
    }
}

// What the board's column scores now: the running battle's round, else today's.
export function scoringNow(race: Race | undefined): Round | undefined {
    return race?.battle ?? race?.today
}

// The rounds still open among those asked for; none until the book's scores arrived.
export function raceOf(
    scores: Score[] | undefined,
    rounds: ReadonlyMap<string, Round>,
    todayRoundId: string | undefined,
    battleRoundId: string | undefined,
): Race | undefined {
    if (!scores) return undefined
    const open = (roundId: string | undefined) => {
        const round = roundId ? rounds.get(roundId) : undefined
        return round && round.endedAt === 0 ? round : undefined
    }
    return {today: open(todayRoundId), battle: open(battleRoundId), scores}
}
