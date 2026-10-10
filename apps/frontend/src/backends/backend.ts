import {BonusReward, BonusRules, Charges, Switches} from "../domain/bonus.ts"
import {QuizOffer, QuizOutcome, QuizQuestion} from "../domain/quiz.ts"

export interface TileClicker {
    clickTile(tileId: number, countryId: string, switches?: Switches): Promise<void>
}

export type Ownerships = {
    bindings: Map<number, string>
    shields: Map<number, number>
}

export interface OwnershipsGetter {
    getCurrentOwnershipsByBatch(
        batchSize: number,
        maxIndex: number,
        callback: (ownerships: Ownerships) => void,
        signal?: AbortSignal,
    ): Promise<void>

    // The flag each locked landmass is locked to.
    getFortresses(signal?: AbortSignal): Promise<Map<number, string>>
}

export type Update = {
    tile: number,
    previousCountry: string | undefined,
    newCountry: string | undefined,
    clicked: boolean,
    shields: number,
}

export interface UpdatesListener {
    listenForUpdates(callback: (update: Update) => void): () => void

    listenForUpdatesBatch(
        callback: (updates: Update[]) => void,
    ): () => void

    // The stream came back after a gap: what happened in it was never sent.
    listenForResumes(callback: () => void): () => void

    listenForFortifications(callback: (fortification: Fortification) => void): () => void
}

export type Fortification = {
    landmass: number
    countryId: string
    tile: number
}

export type BonusOffer = {
    token: string

    seed: number

    reward: BonusReward

    expiresAt: number
}

export type BonusCatch = {
    countryId: string

    quizSubject?: string
}

export type Enclosure = {
    countryId: string
    closingTile: number
    wall: number[]
    filled: number[]
    yours?: boolean
}

export type SpreadClick = {
    countryId: string
    tile: number
    spread: number[]
}

export type BonusHandlers = {
    onOffered: (offer: BonusOffer) => void
    onTaken: (taken: BonusCatch) => void
    onEnclosed: (enclosure: Enclosure) => void
    onSpread: (spread: SpreadClick) => void
    onCharges: (charges: Charges) => void
    onRules: (rules: BonusRules) => void
}

export type ClaimedBonus = {
    reward: BonusReward
    charges: Charges
}

export interface BonusListener {
    listenForBonuses(handlers: BonusHandlers): () => void

    claimBonus(token: string, countryId: string): Promise<ClaimedBonus>
}

export interface QuizMaster {
    listenForQuizzes(onOffered: (offer: QuizOffer) => void): () => void

    openQuiz(token: string): Promise<QuizQuestion>

    answerQuiz(token: string, choice: number, countryId: string): Promise<QuizOutcome>
}

export type GlobePoint = {x: number, y: number, z: number}

export type BombDrop = {
    tile: number | undefined

    point: GlobePoint

    countryId: string

    radius: number

    cleared: number[]

    struck: number[]
}

export interface Bomber {
    listenForBombs(onDropped: (drop: BombDrop) => void): () => void

    dropBomb(target: GlobePoint, countryId: string): Promise<void>
}

export interface Refiller {
    useRefill(countryId: string): Promise<void>
}

export interface Shielder {
    placeShield(tileId: number, countryId: string): Promise<void>
}

export class ShieldRefusedError extends Error {
    constructor(options?: ErrorOptions) {
        super("the tile is not this player's, or holds all the shields it can", options)
        this.name = "ShieldRefusedError"
    }
}

export class BankFullError extends Error {
    constructor(options?: ErrorOptions) {
        super("the click bank is already full", options)
        this.name = "BankFullError"
    }
}

// One error on purpose: telling the reasons apart would help a script guessing tokens.
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
