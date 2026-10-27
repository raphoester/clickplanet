export type TurnKind = "battle" | "plain"

export type Turn = {
    kind: TurnKind
    since?: number
    endsAt?: number
    description: string
}

export const PLAIN_TURN: Turn = {kind: "plain", description: ""}

export type BattleToCome = {
    entryId: string
    startsAt: number
    endsAt: number
    description: string
}

export type TurnListener = {
    onTurn: (turn: Turn) => void
    onBattles: (battles: BattleToCome[]) => void
}

export interface TurnBackend {
    // The turn and the battles to come at once, then each change of either.
    follow(listener: TurnListener): () => void
}
