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

/**
 * Somebody closed a shape with an enclose bonus, and took what was inside.
 *
 * The tiles themselves arrive as ordinary tile updates. This is what lets every
 * client show *why* they changed, which a patch of tiles flipping at once would
 * not say on its own.
 */
export type Enclosure = {
    countryId: string
    /** The click that closed it; one of the wall tiles. */
    closingTile: number
    /** The closer's tiles around the inside: the shape's outline. */
    wall: number[]
    /** The tiles taken, nearest the closing tile first. */
    filled: number[]
    /** Set only when this client closed it: how many shapes its bonus has left. */
    yours?: {shapesLeft: number}
}

export type BonusHandlers = {
    onOffered: (offer: BonusOffer) => void
    onTaken: (taken: BonusCatch) => void
    onEnclosed: (enclosure: Enclosure) => void
}

export interface BonusListener {
    /**
     * Follows the bonus feed on the connection that is already open: the box
     * drawn for this client, every catch on the planet, and every shape closed.
     */
    listenForBonuses(handlers: BonusHandlers): () => void

    /**
     * Redeems a box. Rejects with `BonusLostError` when the server will not
     * honour it — lapsed, already spent, or never this caller's.
     */
    claimBonus(token: string, countryId: string): Promise<BonusReward>
}

/** A direction from the centre of the globe. Need not be unit length. */
export type GlobePoint = {x: number, y: number, z: number}

/**
 * A bomb that landed, anywhere on the planet — broadcast to everyone, the
 * dropper included, so every screen plays the same blast.
 */
export type BombDrop = {
    /** The tile it hit, or undefined for a bomb that fell in the sea. */
    tile: number | undefined

    /** Where to draw it, on the unit sphere. */
    point: GlobePoint

    /** Who dropped it, for the news line. */
    countryId: string

    /** Radians of arc, so the drawing matches what was cleared. */
    radius: number

    /**
     * The tiles the server cleared. Carried here rather than as tile updates so
     * the client can hold them back until the blast hits, instead of the ground
     * going blank before the bomb has landed.
     */
    cleared: number[]
}

export interface Bomber {
    listenForBombs(onDropped: (drop: BombDrop) => void): () => void

    /**
     * Drops the bomb this client won where it was aimed. Whether that is land or
     * sea is the server's call. Rejects with `BonusLostError` when there is none
     * to drop — never won, already dropped, or held too long.
     */
    dropBomb(target: GlobePoint, countryId: string): Promise<void>
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
