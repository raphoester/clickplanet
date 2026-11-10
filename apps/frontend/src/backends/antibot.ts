export interface WarningsListener {
    heldIdentity(): string | undefined

    // A refused token stops it for good: a new token needs a new one.
    listenForWarnings(onWarned: () => void): () => void
}

export interface WarningAcknowledger {
    acknowledgeWarning(): Promise<void>
}
