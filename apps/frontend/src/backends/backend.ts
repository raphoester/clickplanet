export interface TileClicker {
    clickTile(tileId: number, countryId: string): Promise<void>
}

export type Ownerships = {
    bindings: Map<number, string>
}

export interface OwnershipsGetter {
    getCurrentOwnershipsByBatch(
        batchSize: number,
        maxIndex: number,
        callback: (ownerships: Ownerships) => void,
        signal?: AbortSignal,
    ): Promise<void>
}

export type Update = {
    tile: number,
    previousCountry: string | undefined,
    newCountry: string
}

export interface UpdatesListener {
    /**
     * Subscribes to individual updates. Returns the unsubscribe function
     * directly: there is nothing meaningful to await, because a transport that
     * reconnects has no single moment of "connected" to resolve on.
     */
    listenForUpdates(callback: (update: Update) => void): () => void

    listenForUpdatesBatch(
        callback: (updates: Update[]) => void,
    ): () => void
}
