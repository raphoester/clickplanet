import {BonusReward} from "../domain/bonus.ts"

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
    listenForUpdates(callback: (update: Update) => void): () => void

    listenForUpdatesBatch(
        callback: (updates: Update[]) => void,
    ): () => void
}

/**
 * A box the server has put in front of **this** client, and nobody else.
 *
 * The offer is addressed rather than broadcast: the server draws one connected
 * caller and sends the box down that stream. So there is no race to win here and
 * no other client to beat — see the backend's bonus package for why.
 */
export type BonusOffer = {
    /** Single use, and worth nothing to any other caller. */
    token: string

    /** Names the flight path; every client draws the same orbit from it. */
    seed: number

    reward: BonusReward

    /**
     * When the token stops being accepted, on the monotonic clock — the same one
     * `ClickBudget.readAt` is stamped against.
     *
     * Derived from how long the server said was *left*, not from the timestamp
     * it sent: the two machines' wall clocks are unrelated, and a client whose
     * clock is a minute out would otherwise think every box had already lapsed.
     */
    expiresAt: number
}

/** Somebody caught one. Carries no token: it is news, not an offer. */
export type BonusCatch = {
    countryId: string
}

export interface BonusListener {
    /**
     * Follows both halves of the bonus feed on the connection that is already
     * open: the box drawn for this client, and every catch on the planet.
     */
    listenForBonuses(handlers: {
        onOffered: (offer: BonusOffer) => void
        onTaken: (taken: BonusCatch) => void
    }): () => void

    /**
     * Redeems a box. Rejects with `BonusLostError` when the server will not
     * honour it — lapsed, already spent, or never this caller's.
     */
    claimBonus(token: string, countryId: string): Promise<BonusReward>
}

/**
 * The box could not be claimed. Deliberately one error and not four: the server
 * does not say which of the reasons it was, because the difference is what a
 * script guessing tokens would measure.
 */
export class BonusLostError extends Error {
    constructor(options?: ErrorOptions) {
        super("the bonus could not be claimed", options)
        this.name = "BonusLostError"
    }
}

export class RateLimitedError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("too many clicks", options)
        this.name = "RateLimitedError"
    }
}

export class VPNBlockedError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("clicks from VPN addresses are refused", options)
        this.name = "VPNBlockedError"
    }
}
