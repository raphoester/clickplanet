import {BonusReward, BonusRules, Charges, Switches} from "../domain/bonus.ts"
import {QuizOffer, QuizOutcome, QuizQuestion} from "../domain/quiz.ts"

export interface TileClicker {
    /**
     * `switches` are the bonuses the player has switched on: the server spends
     * a spread click or an enclosure on this click only when its switch is on.
     */
    clickTile(tileId: number, countryId: string, switches?: Switches): Promise<void>
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
    /** Undefined when an operator gives a tile back to nobody. */
    newCountry: string | undefined
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

    /**
     * The country the question was about, when the charge was won by answering a quiz rather than
     * by catching a box. Undefined for a box.
     */
    quizSubject?: string
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
    /** Set only when this client closed it. */
    yours?: boolean
}

/**
 * Somebody clicked under a spread bonus. The tiles arrive as tile updates, like
 * an enclosure's; this is what lets every client show why.
 */
export type SpreadClick = {
    countryId: string
    /** The tile clicked. */
    tile: number
    /** The tiles touching it, which the click also took. Empty for a lone island. */
    spread: number[]
}

export type BonusHandlers = {
    onOffered: (offer: BonusOffer) => void
    onTaken: (taken: BonusCatch) => void
    onEnclosed: (enclosure: Enclosure) => void
    onSpread: (spread: SpreadClick) => void
    /** What this player holds now: once it is read, and after every change this client makes or learns of. */
    onCharges: (charges: Charges) => void
    /** How big each charge is, once it is read at load. */
    onRules: (rules: BonusRules) => void
}

/** What a caught box was worth, and what the player holds once it is granted. */
export type ClaimedBonus = {
    reward: BonusReward
    charges: Charges
}

export interface BonusListener {
    /**
     * Follows the bonus feed on the connection that is already open: the box
     * drawn for this client, and every catch, shape closed and spread click on
     * the planet.
     *
     * The charges held and the rules are not on the stream: they are read, and
     * handed to a new listener at once when they already have been.
     */
    listenForBonuses(handlers: BonusHandlers): () => void

    /**
     * Redeems a box. Rejects with `BonusLostError` when the server will not
     * honour it — lapsed, already spent, or never this caller's.
     */
    claimBonus(token: string, countryId: string): Promise<ClaimedBonus>
}

/**
 * The quizzes: a second way to earn one of the same charges, asked rather than caught.
 *
 * Separate from `BonusListener` because it is a separate thing with a separate schedule, and
 * because a client that draws no boxes can still ask questions. The feed is the same connection:
 * a banner is one more case in the one stream envelope.
 *
 * **Two calls to answer one question, and the split is the security model.** `openQuiz` is what
 * starts the server's own five-second clock, so a banner can sit unopened without burning it; and
 * which of the three choices is right is never sent until `answerQuiz` has already been called.
 */
export interface QuizMaster {
    /** Follows the banners on the connection that is already open. Only this client's ever arrive. */
    listenForQuizzes(onOffered: (offer: QuizOffer) => void): () => void

    /**
     * Reads the question and starts its clock. Rejects with `BonusLostError` when the server will
     * not honour the token — lapsed, answered, or never this caller's.
     *
     * Opening twice is safe and is not a second chance: the same question comes back with the same
     * deadline, so a reload shows less time rather than more.
     */
    openQuiz(token: string): Promise<QuizQuestion>

    /**
     * Answers it. A **wrong** answer is not a rejection: it resolves, saying so and saying which one
     * was right. Only a token the server will not honour rejects, with `BonusLostError`.
     */
    answerQuiz(token: string, choice: number, countryId: string): Promise<QuizOutcome>
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
     * Drops the bomb this player holds where it was aimed. Whether that is land
     * or sea is the server's call. Rejects with `BonusLostError` when there is
     * none to drop — never won, already dropped, or held for more than a day.
     */
    dropBomb(target: GlobePoint, countryId: string): Promise<void>
}

export interface Refiller {
    /**
     * Spends the refill this player holds: the click bank is filled to full,
     * and the budget and the charges follow. Rejects with `BankFullError` when
     * the bank is already full, which spends nothing, and with
     * `BonusLostError` when there is no refill to use.
     */
    useRefill(countryId: string): Promise<void>
}

/** A refill used on a full bank: the server refused it and spent nothing. */
export class BankFullError extends Error {
    constructor(options?: ErrorOptions) {
        super("the click bank is already full", options)
        this.name = "BankFullError"
    }
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
