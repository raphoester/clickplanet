import {Season, SeasonTotals} from "../backends/season.ts"
import {BookStep, Round, RoundStanding, Score, Standing} from "../backends/standings.ts"

export const STATS_SHOWN = 10

const DAY_MS = 24 * 60 * 60 * 1000

export type Slide = "table" | "race" | "players" | "battleFlags" | "battlePlayers" | "numbers"

// The Final Battle's two slides only for a season that had one.
export function slidesOf(season: Season): Slide[] {
    return season.finaleRoundId
        ? ["table", "race", "players", "battleFlags", "battlePlayers", "numbers"]
        : ["table", "race", "players", "numbers"]
}

// Every country first in a book's table: level on everything, they won together.
export function firstsOf(table: Score[]): string[] {
    return table.filter((score) => score.rank === 1).map((score) => score.countryCode)
}

// width is the row's value against the first one's, for its bar.
export type Barred<T> = T & {width: number}

// Every country of the table, not ten: the season's ranking is the slide's.
export function seasonFlagRowsOf(table: Score[]): Barred<Score>[] {
    return barred(table, (score) => score.points)
}

export function playerRowsOf(players: Standing[]): Barred<Standing>[] {
    return barred(players.slice(0, STATS_SHOWN), (player) => player.tiles)
}

export type BattleFlagRow = Barred<RoundStanding> & {tiles: number}

export function battleFlagRowsOf(round: Round): BattleFlagRow[] {
    const tiles = new Map(round.takes.map((takes) => [takes.countryCode, takes.tiles]))
    return barred(round.standings.filter(({points}) => points > 0).slice(0, STATS_SHOWN), (standing) => standing.points)
        .map((row) => ({...row, tiles: tiles.get(row.countryCode) ?? 0}))
}

export type BigNumber = {label: string, value: number}

// What the season's reads hold: a read not there yet, or that failed, leaves its numbers out.
export function bigNumbersOf(season: Season, totals: SeasonTotals | undefined, table: Score[] | undefined, finale: Round | undefined): BigNumber[] {
    const numbers: BigNumber[] = []
    if (totals) numbers.push({label: "Clicks", value: totals.takes}, {label: "Players", value: totals.players})
    numbers.push({label: "Days", value: Math.round((season.endsAt - season.startsAt) / DAY_MS)})
    if (totals) {
        numbers.push(
            {label: "Bombs dropped", value: totals.bombs},
            {label: "Shields placed", value: totals.shields},
            {label: "Shield hits", value: totals.strikes},
            {label: "Lands fortified", value: totals.fortifies},
        )
    }
    if (table) numbers.push({label: "Flags that scored", value: table.length})
    if (finale) numbers.push({label: "Final Battle clicks", value: finale.takes.reduce((sum, takes) => sum + takes.tiles, 0)})
    return numbers
}

export const RACE_SHOWN = 8

export type RaceMeasure = "points" | "share"

export type RaceLine = {countryCode: string, values: number[]}

// The flags first at the season's end, each with its value after every round: its points in the book's table, or its
// share of the map at the round's end.
export function raceOf(steps: BookStep[], measure: RaceMeasure, shown = RACE_SHOWN): {lines: RaceLine[], most: number} {
    const valueOf = (step: BookStep, code: string): number => measure === "points"
        ? step.table.find((score) => score.countryCode === code)?.points ?? 0
        : step.results.find((standing) => standing.countryCode === code)?.share ?? 0
    const last = steps.at(-1)
    const codes = last
        ? (measure === "points" ? last.table.map((score) => score.countryCode) : [...last.results].sort((a, b) => b.share - a.share).map((standing) => standing.countryCode))
        : []
    const lines = codes.slice(0, shown).map((countryCode) => ({countryCode, values: steps.map((step) => valueOf(step, countryCode))}))
    return {lines, most: Math.max(measure === "points" ? 1 : 0.01, ...lines.flatMap((line) => line.values))}
}

function barred<T>(rows: T[], value: (row: T) => number): Barred<T>[] {
    const most = Math.max(1, ...rows.map(value))
    return rows.map((row) => ({...row, width: value(row) / most}))
}
