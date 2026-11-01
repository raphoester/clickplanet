import {ClosedRound, RoundStanding} from "../backends/standings.ts"
import {Debrief} from "../backends/turn.ts"
import {Battle} from "./battle.ts"
import {debriefRoundOf} from "./debrief.ts"

export const POINTS_SHOWN = 10

// Seconds before a screen shows its button.
export const BATTLE_DEBRIEF = {over: 2.4, points: 2.2, players: 2.2} as const

export type DebriefScreen = keyof typeof BATTLE_DEBRIEF

// A battle's round plays its debrief when it is the battle the page saw, or when the page saw none.
export function debriefedFor(closed: ClosedRound, seen: Battle | undefined): boolean {
    return !seen || seen.roundId === closed.roundId
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
    if (!seen || seen.roundId !== closed.roundId) return ""
    return seen.description
}

// The battle ended and the gamemaster has not debriefed its round yet: its debrief is on the way.
export function briefAwaited(over: Battle, debrief: Debrief | undefined): boolean {
    return over.roundId !== "" && debriefRoundOf(debrief) !== over.roundId
}

// Scoring may be down: a battle's start waits no longer than this for the day it ended.
export const DAY_BRIEF_WAIT_MS = 10_000

// A battle that is a round starts as a day ends, and that day's debrief, once read, plays before the battle's start.
export function dayBriefAwaited(battle: Battle, closed: ClosedRound | undefined): boolean {
    return battle.roundId !== "" && (closed === undefined || closed.endedAt < battle.since)
}
