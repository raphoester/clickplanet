import {BonusReward, BonusRules, Charges, Switches} from "../domain/bonus.ts"
import {QuizOffer, QuizOutcome, QuizQuestion} from "../domain/quiz.ts"
import {Allegiance} from "../domain/allegiance.ts"
import {Rally} from "../domain/rallies.ts"

export interface TileClicker {
    // cost is what the page counts the click takes from the bank until the server answers: less inside a rally's ring.
    clickTile(tileId: number, countryId: string, switches?: Switches, cost?: number): Promise<void>
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
    // The flags knocked out on the map read.
    knockedOut: readonly string[]
    // The virus that infects each tile of the range a live one infects.
    infected: Map<number, string>
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

    listenForKnockOuts(callback: (knockOut: KnockOut) => void): () => void
}

export type Fortification = {
    landmass: number
    countryId: string
    tile: number
    version: MapVersion
}

// A flag knocked out: another flag took its capital, and every other tile it held went blank.
export type KnockOut = {
    countryId: string
    byCountryId: string
    tile: number
    cleared: number[]
    version: MapVersion
}

// A box in flight: a question mark, since what it holds is drawn when it is caught.
export type BoxOffer = {
    token: string

    seed: number

    expiresAt: number
}

export type Enclosure = {
    countryId: string
    closingTile: number
    wall: number[]
    filled: number[]
    // Each lost a shield and kept its flag.
    struck: {tile: number, shields: number}[]
    version: MapVersion
    yours?: boolean
}

export type SpreadClick = {
    countryId: string
    tile: number
    spread: number[]
}

export type BonusHandlers = {
    onEnclosed: (enclosure: Enclosure) => void
    onVirusSpread: (spread: VirusSpread) => void
    // The mines of the flag the server says the page plays for: read again as each stream opens and the flag changes.
    onMines: (mines: Minefield) => void
    // A mine a player of the page's flag laid: no other flag hears of it.
    onMineLaid: (laid: MineLaid) => void
    onMinesExploded: (explosion: MinesExploded) => void
    // Every rally still on, of every flag: read again as each stream opens.
    onRallies: (rallies: Rally[]) => void
    // A rally a player called, whatever its flag.
    onRallyCalled: (rally: Rally) => void
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
}

// The boxes offered to the page, on a stream of their own. A box caught pays nothing itself: what it held is a grant
// on planet's stream a moment after.
export interface BoxCatcher {
    listenForBoxes(onOffered: (offer: BoxOffer) => void): () => void

    // What the box held, as planet's stream tells it; undefined when the hand had room for none.
    claimBox(token: string): Promise<BonusReward | undefined>
}

// For a charge granted elsewhere, as a mission's reward: the holder tells its listeners again.
export interface ChargesRefresher {
    refreshCharges(): Promise<void>
}

// Bonuses has no stream: its rules are read again when the set in force changes.
export interface GrantRulesRefresher {
    refreshGrantRules(): Promise<void>
}

export interface QuizMaster {
    listenForQuizzes(onOffered: (offer: QuizOffer) => void): () => void

    openQuiz(token: string): Promise<QuizQuestion>

    // Right or wrong: what a right answer won comes a moment after, as a reward.
    answerQuiz(token: string, choice: number, countryId: string): Promise<QuizOutcome>

    listenForRewards(onRewarded: (reward: BonusReward) => void): () => void
}

// Each charge a grant gave the page's account, as planet's stream tells it: a right answer's, a mission's.
export interface GrantSource {
    listenForGrants(onGranted: (rewards: BonusReward[]) => void): () => void
}

// Each session token a call of the page carried: a stream that reads its token only as it opens follows it.
export interface SessionSource {
    listenForSessions(onSession: (token: string) => void): () => void
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

// One use or tick of a virus: no tile update comes for its tiles.
export type VirusSpread = {
    virus: string

    // The flag it takes tiles for, and the flag of the tile it was used on: it spreads into every enemy flag's land.
    countryId: string
    victimCountryId: string

    // Each is countryId's now, and infected.
    taken: number[]

    // Each lost a shield and kept its flag.
    struck: {tile: number, shields: number}[]

    // It spreads no more: every tile it infects is cured.
    ended: boolean

    version: MapVersion
}

export interface Infector {
    useVirus(tileId: number, countryId: string): Promise<void>
}

export type Minefield = {
    // The flag the server says the page plays for; empty for none.
    countryId: string
    tiles: number[]
    version: MapVersion
}

// A mine laid changes no tile: its tile holds it while the tile has not changed since version.
export type MineLaid = {
    countryId: string
    tile: number
    version: MapVersion
}

// The mines a take set off, every mine a blast reached going off too, in order: no tile update comes for their tiles.
export type MinesExploded = {
    blasts: MineBlast[]
    version: MapVersion
}

export type MineBlast = {
    // The tile the mine was on.
    tile: number
    // The mine's flag, whose each tile taken is now.
    countryId: string
    taken: number[]
}

export interface Miner {
    layMine(tileId: number, countryId: string): Promise<void>
}

export interface Rallier {
    callRally(tileId: number, countryId: string): Promise<Rally>
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

    refreshAllegiance(): Promise<void>
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

// The tile was no enemy tile, or held a shield: the virus stays in the hand.
export class VirusRefusedError extends Error {
    constructor(options?: ErrorOptions) {
        super("a virus takes only an enemy tile with no shield", options)
        this.name = "VirusRefusedError"
    }
}

// The tile was not the flag's, held a mine already, or was a capital the knock-out plays on: the mine stays in the hand.
export class MineRefusedError extends Error {
    constructor(options?: ErrorOptions) {
        super("a mine goes only on a tile of your flag with no mine", options)
        this.name = "MineRefusedError"
    }
}

// A rally the caller called is still on: the rally stays in the hand.
export class RallyRefusedError extends Error {
    constructor(options?: ErrorOptions) {
        super("a rally of yours is still on", options)
        this.name = "RallyRefusedError"
    }
}

export class BankFullError extends Error {
    constructor(options?: ErrorOptions) {
        super("the click bank is already full", options)
        this.name = "BankFullError"
    }
}

// The rules in force hold bombs off: the bomb stays held.
export class BombRefusedError extends Error {
    constructor(options?: ErrorOptions) {
        super("bombs are off", options)
        this.name = "BombRefusedError"
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

// The rules in force freeze the map: it takes no play until they change.
export class FrozenError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("the map is frozen", options)
        this.name = "FrozenError"
    }
}

// While the reach plays, a tile next to none of the flag's land and at the far end of no bridge from it.
export class OutOfReachError extends Error {
    constructor(options?: {cause?: unknown}) {
        super("the tile is out of the flag's reach", options)
        this.name = "OutOfReachError"
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
