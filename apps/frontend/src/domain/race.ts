import {Race, Round, Score} from "../backends/standings.ts"
import {Countries, Country} from "./countries.ts"
import {LeaderboardEntry} from "./leaderboard.ts"

export type CountryOrder = "points" | "tiles" | "gain" | "today"

export type CountryLine = {
    country: Country
    tiles: number
    points: number
    today: number
    // The tiles gained since the round that scores now opened; none while no round runs.
    gain: number | undefined
    // Its place in that round at the round's last read; none for a country the round did not place.
    place: number | undefined
    // Its rank on the tiles held and on the gain among the countries that hold ground now; none while no round runs
    // or with no ground. A day places the sum of the two.
    ranks: Ranks | undefined
}

export type Ranks = {tiles: number, gain: number}

export function shownOrder(race: Race | undefined, picked: CountryOrder = "points"): CountryOrder {
    if (!race) return "tiles"
    if ((picked === "gain" || picked === "today") && !scoringNow(race)) return "points"
    return picked
}

export function countryLines(
    leaderboard: readonly LeaderboardEntry[],
    race: Race | undefined,
    order: CountryOrder,
): CountryLine[] {
    const now = scoringNow(race)
    const today = new Map(now?.standings.map((standing) => [standing.countryCode, standing]))
    const start = new Map(now?.start.map((held) => [held.countryCode, held.tiles]))
    const points = new Map(race?.scores.map((score) => [score.countryCode, score.points]))
    const line = (entry: LeaderboardEntry): CountryLine => ({
        ...entry,
        points: points.get(entry.country.code) ?? 0,
        today: today.get(entry.country.code)?.points ?? 0,
        gain: now ? entry.tiles - (start.get(entry.country.code) ?? 0) : undefined,
        place: today.get(entry.country.code)?.rank,
        ranks: undefined,
    })

    const lines = ranked(leaderboard.map(line))
    if (order === "tiles") return lines

    const held = new Set(leaderboard.map((entry) => entry.country.code))
    for (const code of new Set([...points.keys(), ...today.keys()])) {
        const country = Countries.get(code)
        if (!held.has(code) && country) lines.push(line({country, tiles: 0}))
    }
    return lines
        .filter((l) => l.tiles > 0 || l.points + l.today > 0)
        .sort((a, b) => ahead(a, b, order) || b.tiles - a.tiles || a.country.code.localeCompare(b.country.code))
}

function ahead(a: CountryLine, b: CountryLine, order: "points" | "gain" | "today"): number {
    switch (order) {
        case "points":
            return b.points - a.points
        case "gain":
            return (b.gain ?? 0) - (a.gain ?? 0)
        case "today":
            return (a.place ?? UNPLACED) - (b.place ?? UNPLACED)
    }
}

const UNPLACED = Number.MAX_SAFE_INTEGER

// As scoring ranks them: among the countries that hold ground, the most first, a tie sharing its rank (1, 1, 3).
function ranked(lines: CountryLine[]): CountryLine[] {
    const holding = lines.filter((line) => line.tiles > 0)
    const tilesRank = rankAmong(holding.map((line) => line.tiles))
    const gainRank = rankAmong(holding.map((line) => line.gain ?? 0))
    return lines.map((line) => line.tiles > 0 && line.gain !== undefined
        ? {...line, ranks: {tiles: tilesRank(line.tiles), gain: gainRank(line.gain)}}
        : line)
}

function rankAmong(values: number[]): (value: number) => number {
    const sorted = [...values].sort((a, b) => b - a)
    return (value) => sorted.indexOf(value) + 1
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
