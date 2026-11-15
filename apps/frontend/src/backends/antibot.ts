export interface WarningsListener {
    heldIdentity(): string | undefined

    // A refused token stops it for good: a new token needs a new one. onNoticed hears the notice that another account
    // played from the player's connection at the same time, which bans nothing.
    listenForWarnings(onWarned: () => void, onNoticed: () => void): () => void
}

export interface WarningAcknowledger {
    acknowledgeWarning(): Promise<void>
}

export interface NoticeAcknowledger {
    acknowledgeNotice(): Promise<void>
}
