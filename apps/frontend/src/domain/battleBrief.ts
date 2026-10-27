import {ClosedRound, Race, RoundStanding} from "../backends/standings.ts"
import {Battle} from "./battle.ts"

export const POINTS_SHOWN = 10

// Seconds before a screen shows its button.
export const BATTLE_DEBRIEF = {over: 2.4, points: 2.2, players: 2.2} as const

export type DebriefScreen = keyof typeof BATTLE_DEBRIEF

// A round of its own plays its debrief when it is the battle the page saw, or when the page saw none.
export function debriefedFor(closed: ClosedRound, seen: Battle | undefined): boolean {
    return !seen || seen.entryId === closed.entryId
}

// The battle finished, then only the screens with something to show.
export function debriefScreensOf(closed: ClosedRound): DebriefScreen[] {
    const screens: DebriefScreen[] = ["over"]
    if (closed.standings.some(({points}) => points > 0)) screens.push("points")
    if (closed.players.length > 0) screens.push("players")
    return screens
}

export type PointsRow = RoundStanding & {won: number, width: number, mine: boolean}

// won is the points before the round's times.
export function pointsRowsOf(closed: ClosedRound, countryCode: string, shown = POINTS_SHOWN): PointsRow[] {
    const scored = closed.standings.filter(({points}) => points > 0)
    const most = Math.max(1, ...scored.map(({points}) => points))
    const mine = scored.findIndex((standing) => standing.countryCode === countryCode)
    return scored
        .filter((_, i) => i < shown || i === mine)
        .map((standing) => ({
            ...standing,
            won: Math.round(standing.points / Math.max(1, closed.times)),
            width: standing.points / most,
            mine: standing.countryCode === countryCode,
        }))
}

// The battle's words, when the page saw this battle play.
export function briefDescriptionOf(seen: Battle | undefined, closed: ClosedRound): string {
    if (!seen || seen.entryId !== closed.entryId) return ""
    return seen.description
}

// The battle ended with its round of its own the race still shows in progress: its debrief is on the way.
export function briefAwaited(over: Battle, race: Race | undefined): boolean {
    return race?.round?.own === true && race.round.entryId === over.entryId
}
