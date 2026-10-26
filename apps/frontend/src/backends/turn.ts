export type TurnKind = "battle" | "plain"

export type Turn = {
    kind: TurnKind
    since?: number
    endsAt?: number
    description: string
}

export const PLAIN_TURN: Turn = {kind: "plain", description: ""}

export interface TurnBackend {
    turn(): Promise<Turn>

    // Tells when what it answers may have changed, so the caller reads again.
    changes(onChange: () => void): () => void
}
