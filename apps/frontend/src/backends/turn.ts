import {Seasons} from "./season.ts"

export type TurnKind = "battle" | "plain"

export type Turn = {
    kind: TurnKind
    since?: number
    endsAt?: number
    description: string
    // The running entry's id, empty when none runs.
    entryId: string
    // The running battle's round in scoring; empty when no battle runs.
    roundId: string
    times: number
}

export const PLAIN_TURN: Turn = {kind: "plain", description: "", entryId: "", roundId: "", times: 0}

export type BattleToCome = {
    entryId: string
    startsAt: number
    endsAt: number
    description: string
    times: number
    endsSeason: boolean
    roundId: string
}

export type BattleEnded = {
    kind: "battle"
    season: number
    number: number
    entryId: string
    times: number
    endsSeason: boolean
    roundId: string
    endedAt: number
}

// What scoring just scored, in the gamemaster's words: the numbers are read from scoring by its ids.
export type Debrief =
    | {kind: "day", season: number, number: number, roundId: string, endedAt: number}
    | BattleEnded
    | {kind: "season", season: number, bookId: string, endedAt: number, finaleRoundId: string, finaleBattle?: BattleEnded}

export type TurnListener = {
    onTurn: (turn: Turn) => void
    onBattles: (battles: BattleToCome[]) => void
    onSeasons: (seasons: Seasons) => void
    // The last debrief; none before any.
    onDebrief: (debrief: Debrief | undefined) => void
}

export interface TurnBackend {
    // The turn, the battles to come, the seasons and the last debrief at once, then each change of any.
    follow(listener: TurnListener): () => void
}
