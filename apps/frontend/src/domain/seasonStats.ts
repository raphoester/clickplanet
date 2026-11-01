import {Season, SeasonTotals} from "../backends/season.ts"
import {Round, RoundStanding, Score, Standing} from "../backends/standings.ts"

export const STATS_SHOWN = 10

const DAY_MS = 24 * 60 * 60 * 1000

export type Slide = "flags" | "players" | "battleFlags" | "battlePlayers" | "numbers"

// The Final Battle's two slides only for a season that had one.
export function slidesOf(season: Season): Slide[] {
    return season.finaleRoundId
        ? ["flags", "players", "battleFlags", "battlePlayers", "numbers"]
        : ["flags", "players", "numbers"]
}

// width is the row's value against the first one's, for its bar.
export type Barred<T> = T & {width: number}

export function seasonFlagRowsOf(table: Score[]): Barred<Score>[] {
    return barred(table.slice(0, STATS_SHOWN), (score) => score.points)
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

function barred<T>(rows: T[], value: (row: T) => number): Barred<T>[] {
    const most = Math.max(1, ...rows.map(value))
    return rows.map((row) => ({...row, width: value(row) / most}))
}
