import {ClosedRound, Race, RoundTakes} from "../backends/standings.ts"
import {Battle} from "./battle.ts"
import {podiumOf, ROUND_REVEAL} from "./roundReveal.ts"

export const TAKES_SHOWN = 8

// Seconds a stage shows before the next one; the last one waits settle before Close.
export const BATTLE_BRIEF = {podium: ROUND_REVEAL.podium, takes: 4.2, players: 4.2, settle: 1.8} as const

export type BriefStage = "podium" | "takes" | "players"

// Only the stages with something to show; the head alone when none has.
export function briefStagesOf(closed: ClosedRound): BriefStage[] {
    const stages: BriefStage[] = []
    if (podiumOf(closed).length > 0) stages.push("podium")
    if (closed.takes.some(({tiles}) => tiles > 0)) stages.push("takes")
    if (closed.players.length > 0) stages.push("players")
    return stages.length > 0 ? stages : ["podium"]
}

export type TakesRow = RoundTakes & {width: number, mine: boolean}

export function takesRowsOf(closed: ClosedRound, countryCode: string, shown = TAKES_SHOWN): TakesRow[] {
    const taken = closed.takes.filter(({tiles}) => tiles > 0)
    const most = Math.max(1, ...taken.map(({tiles}) => tiles))
    const mine = taken.findIndex((takes) => takes.countryCode === countryCode)
    return taken
        .filter((_, i) => i < shown || i === mine)
        .map((takes) => ({...takes, width: takes.tiles / most, mine: takes.countryCode === countryCode}))
}

// The page read the battle's words while it played. Its window is the round's: one end or one start is enough, since a
// round of its own is cut at the season's end.
export function briefDescriptionOf(seen: Battle | undefined, closed: ClosedRound): string {
    if (!seen || (seen.endsAt !== closed.endedAt && seen.since !== closed.startedAt)) return ""
    return seen.description
}

// The battle ended with a round of its own the race still shows in progress: its brief is on the way.
export function briefAwaited(over: Battle, race: Race | undefined): boolean {
    return race?.round?.own === true && race.round.endsAt <= over.endsAt
}
