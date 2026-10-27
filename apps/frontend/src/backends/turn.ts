import {Seasons} from "./season.ts"

export type TurnKind = "battle" | "plain"

export type Turn = {
    kind: TurnKind
    since?: number
    endsAt?: number
    description: string
    // The running entry's id, empty when none runs: the closed round of its own repeats it.
    entryId: string
}

export const PLAIN_TURN: Turn = {kind: "plain", description: "", entryId: ""}

export type BattleToCome = {
    entryId: string
    startsAt: number
    endsAt: number
    description: string
}

export type TurnListener = {
    onTurn: (turn: Turn) => void
    onBattles: (battles: BattleToCome[]) => void
    onSeasons: (seasons: Seasons) => void
}

export interface TurnBackend {
    // The turn, the battles to come and the seasons at once, then each change of any.
    follow(listener: TurnListener): () => void
}
