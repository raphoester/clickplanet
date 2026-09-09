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

/**
 * The server refused a click because too many arrived from this address — the
 * backend keeps a token bucket per IP on the Click RPC, and answers a spent
 * bucket with `resource_exhausted`.
 *
 * It is named here, beside the interface it comes out of, because it is the one
 * click failure the player is meant to see rather than a transport fault: the
 * app layer shows a dialog for it, and must not have to know what a Connect
 * code is to recognise it.
 */
export class RateLimitedError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("too many clicks", options)
        this.name = "RateLimitedError"
    }
}
