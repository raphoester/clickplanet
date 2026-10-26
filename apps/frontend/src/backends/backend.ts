import {BonusReward, BonusRules, Charges, Switches} from "../domain/bonus.ts"
import {QuizOffer, QuizOutcome, QuizQuestion} from "../domain/quiz.ts"
import {Allegiance} from "../domain/allegiance.ts"

export interface TileClicker {
    clickTile(tileId: number, countryId: string, switches?: Switches): Promise<void>
}

// Every write of the map counts one in seq; epoch is the second the server started, when seq started from 0.
export type MapVersion = {
    epoch: number
    seq: number
}

// Both ends inclusive.
export type TileRange = {
    start: number
    end: number
}

export type Ownerships = {
    range: TileRange
    version: MapVersion
    bindings: Map<number, string>
    shields: Map<number, number>
}

// A read lists every row; a change lists the rows that changed, a flag that left the map at 0.
export type Counts = {
    version: MapVersion
    countries: ReadonlyMap<string, number>
    // Each row whole: a flag it does not list holds none of the landmass.
    landmasses: ReadonlyMap<number, ReadonlyMap<string, number>>
}

export interface OwnershipsGetter {
    getOwnerships(
        ranges: readonly TileRange[],
        callback: (ownerships: Ownerships) => void,
        signal?: AbortSignal,
    ): Promise<void>

    getHoldings(signal?: AbortSignal): Promise<Counts>

    // The flag each locked landmass is locked to.
    getFortresses(signal?: AbortSignal): Promise<Map<number, string>>
}

export type TileHolder = {
    accountId: string
    countryCode: string
    takenAt: number
}

export interface TileHolders {
    tileHolder(tile: number, signal?: AbortSignal): Promise<TileHolder | undefined>
}

// The tiles of a landmass one account's takes hold for one flag.
export type LandmassHolder = {
    accountId: string
    countryCode: string
    tiles: number
}

export type LandmassFortification = {
    accountId: string
    countryCode: string
    at: number
}

export type LandmassState = {
    holders: LandmassHolder[]
    lastFortification: LandmassFortification | undefined
}

export interface LandmassReader {
    landmass(landmass: number, signal?: AbortSignal): Promise<LandmassState>
}

export type Update = {
    tile: number,
    previousCountry: string | undefined,
    newCountry: string | undefined,
    clicked: boolean,
    shields: number,
    version: MapVersion,
}

export interface UpdatesListener {
    listenForUpdates(callback: (update: Update) => void): () => void

    listenForUpdatesBatch(
        callback: (updates: Update[]) => void,
    ): () => void

    // Each time the stream opens: every change after this version will follow. A listener that comes late hears at once the version the stream already passed.
    listenForOpenings(callback: (version: MapVersion) => void): () => void

    listenForHoldings(callback: (change: Counts) => void): () => void

    listenForFortifications(callback: (fortification: Fortification) => void): () => void
}

export type Fortification = {
    landmass: number
    countryId: string
    tile: number
    version: MapVersion
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

// For a charge granted elsewhere, as a mission's reward: the holder tells its listeners again.
export interface ChargesRefresher {
    refreshCharges(): Promise<void>
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

    // Each lost a shield and kept its flag.
    struck: {tile: number, shields: number}[]

    version: MapVersion
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

export interface AllegianceSource {
    listenForAllegiance(callback: (allegiance: Allegiance | undefined) => void): () => void

    pledgeAllegiance(countryId: string): Promise<void>
}

export class AllegianceLockedError extends Error {
    constructor(readonly allegiance: Allegiance, options?: ErrorOptions) {
        super(`this player plays for ${allegiance.countryId} for now`, options)
        this.name = "AllegianceLockedError"
    }
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

export class BannedError extends Error {
    readonly until: number

    constructor(until: number, options?: {cause?: unknown}) {
        super("the account or its network is banned", options)
        this.name = "BannedError"
        this.until = until
    }
}
