import {
    BankFullError,
    BombDrop,
    Bomber,
    BonusHandlers,
    BonusListener,
    BonusLostError,
    BonusOffer,
    ClaimedBonus,
    ShieldRefusedError,
    GlobePoint,
    Enclosure,
    Ownerships,
    OwnershipsGetter,
    QuizMaster,
    RateLimitedError,
    Refiller,
    Shielder,
    TileClicker,
    Update,
    UpdatesListener,
    VPNBlockedError,
} from "./backend.ts";
import {ALL_OFF, BonusReward, BonusRules, Charges, NO_CHARGES, Switches} from "../domain/bonus.ts";
import {QuizOffer, QuizOutcome, QuizQuestion} from "../domain/quiz.ts";
import {ClickBudget, ClickBudgetSource, ClickPrice, now as budgetNow, SharedBy} from "./clickBudget.ts";
import {SessionUnavailableError} from "./session.ts";
import {v4 as UUIDv4} from 'uuid';
import {Countries} from "../domain/countries.ts";
import {nearestTile, tilesWithin} from "../domain/blast.ts";

const TILE_COUNT = 257_000

const CLICKS_PER_SECOND = 0.2
const CLICK_BURST = 60

const TOLL_STEPS = [
    {share: 0.10, slowdown: 1.5},
    {share: 0.20, slowdown: 2.5},
    {share: 0.30, slowdown: 4},
]

const BONUS_EVERY_MS = 20_000
const BONUS_OFFER_TTL_MS = 15_000
const SPREAD_CLICKS = 8
const SPREAD_PER_BOX = 4
const ENCLOSURES = 3
const ENCLOSURES_PER_BOX = 3
const SHIELDS = 30
const SHIELDS_MIN_PER_BOX = 5
const SHIELDS_PER_BOX = 20
const TILE_SHIELDS = 10
const BONUS_KINDS: BonusReward["kind"][] = [
    "refill", "refill", "refill", "refill", "refill",
    "spreadClicks", "spreadClicks",
    "bomb",
    "encloseClicks", "encloseClicks",
    "shields", "shields",
]

const BOMB_RADIUS = 0.032

const SEA_REACH = 0.004

const SPREAD_REACH = 0.0052

const QUIZ_EVERY_MS = 30_000
const QUIZ_OFFER_TTL_MS = 25_000
const QUIZ_ANSWER_MS = 5_000

const FAKE_QUIZZES: {subject: string, text: string, choices: string[], correct: number}[] = [
    {subject: "ee", text: "What is the capital of Estonia?", choices: ["Riga", "Tallinn", "Vilnius"], correct: 1},
    {subject: "np", text: "Which of these shares a border with Nepal?", choices: ["China", "Pakistan", "Myanmar"], correct: 0},
    {subject: "br", text: "How is the capital of Brazil spelled?", choices: ["Brazilia", "Brasilia City", "Brasília"], correct: 2},
]

const BOT_BOMB_EVERY_MS = 25_000

const BOT_CLICKS_PER_SECOND = 4
const ENCLOSE_MAX_TILES = 25

const RULES: BonusRules = {
    blastRadius: BOMB_RADIUS,
    enclosureMaxTiles: ENCLOSE_MAX_TILES,
    spreadClicks: SPREAD_CLICKS,
    enclosures: ENCLOSURES,
    shields: SHIELDS,
    tileShields: TILE_SHIELDS,
    toll: TOLL_STEPS,
}

export type FakeBackendOptions = {
    vpnBlocked?: boolean
    sessionUnavailable?: boolean
    tilePositions?: () => Promise<Float32Array>
}

export class FakeBackend implements TileClicker, OwnershipsGetter, UpdatesListener, ClickBudgetSource, BonusListener, QuizMaster, Bomber, Refiller, Shielder {
    private tileBindings: Map<number, string> = new Map()
    private tileCounts: Map<string, number> = new Map()
    private budgetCountry = ""
    private updateListeners: Map<string, (update: Update) => void> = new Map()
    private pendingUpdates: Update[] = []
    private updateBatchCallbacks: Map<string, (update: Update[]) => void> = new Map()
    private budgetCallbacks: Map<string, (budget: ClickBudget) => void> = new Map()
    private bonusCallbacks: Map<string, BonusHandlers> = new Map()
    private bombCallbacks: Map<string, (drop: BombDrop) => void> = new Map()
    private quizCallbacks: Map<string, (offer: QuizOffer) => void> = new Map()
    private shields: Map<number, number> = new Map()

    private charges: Charges = NO_CHARGES
    private positions: Promise<Float32Array> | undefined
    private readonly tilePositions: (() => Promise<Float32Array>) | undefined

    private offered: BonusOffer | undefined

    private quiz: {token: string, expiresAt: number, asked: typeof FAKE_QUIZZES[number], deadline?: number} | undefined

    private readonly timers: ReturnType<typeof setInterval>[] = []
    private tokens = CLICK_BURST
    private lastRefillMs = Date.now()
    private pace = 1
    private sharedWith: SharedBy | undefined
    private shapesClose = true
    private readonly vpnBlocked: boolean
    private readonly sessionUnavailable: boolean

    constructor(batchUpdateDurationMs: number, options: FakeBackendOptions = {}) {
        this.vpnBlocked = options.vpnBlocked ?? false
        this.sessionUnavailable = options.sessionUnavailable ?? false
        this.tilePositions = options.tilePositions

        for (let i = 1; i <= TILE_COUNT; i++) {
            this.tileBindings.set(i, "fr")
        }
        this.tileCounts.set("fr", TILE_COUNT)

        this.listenForUpdates((update) => {
            this.pendingUpdates.push(update)
        })

        this.timers.push(setInterval(() => this.flushUpdates(), batchUpdateDurationMs))

        const offerable = this.tilePositions ? BONUS_KINDS : BONUS_KINDS.filter((kind) => kind !== "bomb")

        this.timers.push(setInterval(() => {
            const kinds = offerable.filter((kind) => !this.full(kind))
            if (kinds.length === 0) return

            const offer: BonusOffer = {
                token: UUIDv4(),
                seed: Math.floor(Math.random() * 0xffffffff),
                reward: rewardOfKind(kinds[Math.floor(Math.random() * kinds.length)]),
                expiresAt: budgetNow() + BONUS_OFFER_TTL_MS,
            }

            this.offered = offer
            this.bonusCallbacks.forEach(handlers => handlers.onOffered(offer))
        }, BONUS_EVERY_MS))

        this.timers.push(setInterval(() => {
            if (this.quiz || offerable.every((kind) => this.full(kind))) return

            const asked = FAKE_QUIZZES[Math.floor(Math.random() * FAKE_QUIZZES.length)]
            this.quiz = {token: UUIDv4(), expiresAt: budgetNow() + QUIZ_OFFER_TTL_MS, asked}

            const offer: QuizOffer = {token: this.quiz.token, expiresAt: this.quiz.expiresAt}
            this.quizCallbacks.forEach(callback => callback(offer))
        }, QUIZ_EVERY_MS))

        const codes = [...Countries.keys()]

        if (this.tilePositions) {
            this.timers.push(setInterval(() => {
                const tile = Math.floor(Math.random() * TILE_COUNT) + 1
                void this.botBomb(tile, codes[Math.floor(Math.random() * codes.length)])
            }, BOT_BOMB_EVERY_MS))
        }

        const runs = codes.map(() => ({tile: Math.floor(Math.random() * TILE_COUNT), gap: 1 + Math.floor(Math.random() * 100)}))
        this.timers.push(setInterval(() => {
            const index = Math.floor(Math.random() * codes.length)
            const run = runs[index]
            run.tile = (run.tile + run.gap) % TILE_COUNT + 1
            this.applyClick(run.tile, codes[index])
        }, 1000 / BOT_CLICKS_PER_SECOND))
    }

    private flushUpdates() {
        if (this.pendingUpdates.length === 0) return
        const updates = this.pendingUpdates
        this.pendingUpdates = []
        this.updateBatchCallbacks.forEach(callback => callback(updates))
    }

    public close() {
        this.timers.forEach(clearInterval)
        this.timers.length = 0
        this.updateListeners.clear()
        this.updateBatchCallbacks.clear()
        this.budgetCallbacks.clear()
        this.bombCallbacks.clear()
    }

    public async clickTile(tileId: number, countryId: string, switches: Switches = ALL_OFF) {
        if (switches.spread && switches.enclose) throw new Error("spread and enclose switched on together")
        if (this.sessionUnavailable) throw new SessionUnavailableError()
        if (this.vpnBlocked) throw new VPNBlockedError()

        const allowed = this.allow(countryId)
        this.reportBudget()

        if (!allowed) throw new RateLimitedError()
        this.applyClick(tileId, countryId)
        if (switches.enclose && this.shapesClose) this.pretendToEnclose(tileId, countryId)
        if (switches.spread) this.announceBonusClick(tileId, countryId)
    }

    private announceBonusClick(tileId: number, countryId: string) {
        if (this.charges.spreadClicksLeft > 0) {
            this.hold({...this.charges, spreadClicksLeft: this.charges.spreadClicksLeft - 1})
            void this.botSpread(tileId, countryId)
        }
    }

    private full(kind: BonusReward["kind"]): boolean {
        switch (kind) {
            case "refill":
                return this.charges.refill
            case "bomb":
                return this.charges.bomb
            case "encloseClicks":
                return this.charges.enclosures >= ENCLOSURES
            case "spreadClicks":
                return this.charges.spreadClicksLeft >= SPREAD_CLICKS
            case "shields":
                return this.charges.shields >= SHIELDS
        }
    }

    private hold(charges: Charges) {
        this.charges = charges
        this.bonusCallbacks.forEach(handlers => handlers.onCharges(charges))
    }

    private grant(kind: BonusReward["kind"]): ClaimedBonus {
        const held = this.charges
        switch (kind) {
            case "refill":
                this.hold({...held, refill: true})
                return {reward: rewardOfKind(kind), charges: this.charges}
            case "bomb":
                this.hold({...held, bomb: true})
                return {reward: rewardOfKind(kind), charges: this.charges}
            case "encloseClicks": {
                const enclosures = Math.min(held.enclosures + drawUpTo(ENCLOSURES_PER_BOX), ENCLOSURES)
                this.hold({...held, enclosures})
                return {reward: rewardOfKind(kind, enclosures - held.enclosures), charges: this.charges}
            }
            case "spreadClicks": {
                const spreadClicksLeft = Math.min(held.spreadClicksLeft + drawUpTo(SPREAD_PER_BOX), SPREAD_CLICKS)
                this.hold({...held, spreadClicksLeft})
                return {reward: rewardOfKind(kind, spreadClicksLeft - held.spreadClicksLeft), charges: this.charges}
            }
            case "shields": {
                const shields = Math.min(held.shields + drawBetween(SHIELDS_MIN_PER_BOX, SHIELDS_PER_BOX), SHIELDS)
                this.hold({...held, shields})
                return {reward: rewardOfKind(kind, shields - held.shields), charges: this.charges}
            }
        }
    }

    public async useRefill(): Promise<void> {
        if (!this.charges.refill) throw new BonusLostError()

        this.refill()
        if (this.tokens >= CLICK_BURST) throw new BankFullError()

        this.tokens = CLICK_BURST
        this.hold({...this.charges, refill: false})
        this.reportBudget()
    }

    public async botSpread(tile: number, countryId: string) {
        if (!this.tilePositions) return
        const positions = await this.loadPositions()

        const spread = tilesWithin(positions, tile, SPREAD_REACH).filter(id => id !== tile)
        this.applyClick(tile, countryId)
        spread.forEach(id => this.applyClick(id, countryId, false))
        this.bonusCallbacks.forEach(handlers => handlers.onSpread({countryId, tile, spread}))
    }

    public grantBonus(kind: Exclude<BonusReward["kind"], "bomb">): ClaimedBonus {
        return this.grant(kind)
    }

    private pretendToEnclose(tileId: number, countryId: string) {
        if (this.charges.enclosures === 0) return

        const wall = [0, 1, 2, 3, 4, 5].map(step => tileId + step).filter(id => id <= TILE_COUNT)
        const filled = [6, 7, 8].map(step => tileId + step).filter(id => id <= TILE_COUNT)
        filled.forEach(id => this.applyClick(id, countryId, false))

        this.hold({...this.charges, enclosures: this.charges.enclosures - 1})

        const enclosure: Enclosure = {countryId, closingTile: tileId, wall, filled, yours: true}
        this.bonusCallbacks.forEach(handlers => handlers.onEnclosed(enclosure))
    }

    public watchClickBudget(callback: (budget: ClickBudget) => void): () => void {
        const id = UUIDv4()
        this.budgetCallbacks.set(id, callback)
        callback(this.budget())
        return () => this.budgetCallbacks.delete(id)
    }

    private reportBudget() {
        const budget = this.budget()
        this.budgetCallbacks.forEach(callback => callback(budget))
    }

    public closeShapes(close: boolean): void {
        this.shapesClose = close
    }

    public shareClicks(sharedWith?: SharedBy): void {
        this.sharedWith = sharedWith
        this.reportBudget()
    }

    public priceFor(countryId: string): void {
        this.budgetCountry = countryId
        this.reportBudget()
    }

    private budget(): ClickBudget {
        this.refill()

        return {
            tokens: this.tokens,
            capacity: CLICK_BURST,
            perSecond: CLICKS_PER_SECOND * this.pace,
            price: this.price(this.budgetCountry),
            sharedWith: this.sharedWith,
            readAt: budgetNow(),
        }
    }

    private price(countryId: string): ClickPrice {
        const share = (this.tileCounts.get(countryId) ?? 0) / TILE_COUNT

        let slowdown = 1
        for (const step of TOLL_STEPS) {
            if (share < step.share) return {slowdown, share, next: step}
            slowdown = step.slowdown
        }

        return {slowdown, share}
    }

    private applyClick(tileId: number, countryId: string, clicked = true) {
        const prev = this.tileBindings.get(tileId)
        if (prev === countryId) return

        const shields = this.shields.get(tileId) ?? 0
        if (prev !== undefined && shields > 0) {
            this.shield(tileId, prev, shields - 1, clicked)
            return
        }

        this.tileBindings.set(tileId, countryId)
        this.shields.delete(tileId)
        this.count(prev, -1)
        this.count(countryId, 1)
        this.updateListeners.forEach(l => l({
            tile: tileId,
            previousCountry: prev,
            newCountry: countryId,
            clicked,
            shields: 0,
        }))
    }

    private shield(tile: number, owner: string, shields: number, clicked: boolean) {
        if (shields > 0) this.shields.set(tile, shields)
        else this.shields.delete(tile)

        this.updateListeners.forEach(l => l({tile, previousCountry: owner, newCountry: owner, clicked, shields}))
    }

    private strike(tile: number): boolean {
        const shields = this.shields.get(tile) ?? 0
        if (shields === 0) return false

        if (shields > 1) this.shields.set(tile, shields - 1)
        else this.shields.delete(tile)
        return true
    }

    public async placeShield(tileId: number, countryId: string): Promise<void> {
        if (this.sessionUnavailable) throw new SessionUnavailableError()
        if (this.charges.shields === 0) throw new BonusLostError()
        if (!this.botShield(tileId, countryId)) throw new ShieldRefusedError()

        this.hold({...this.charges, shields: this.charges.shields - 1})
    }

    public botShield(tile: number, countryId: string): boolean {
        const shields = this.shields.get(tile) ?? 0
        if (this.tileBindings.get(tile) !== countryId || shields >= TILE_SHIELDS) return false

        this.shield(tile, countryId, shields + 1, false)
        return true
    }

    private count(countryId: string | undefined, by: number) {
        if (countryId) this.tileCounts.set(countryId, (this.tileCounts.get(countryId) ?? 0) + by)
    }

    private allow(countryId: string): boolean {
        this.refill()

        this.pace = 1 / this.price(countryId).slowdown
        if (this.tokens < 1) return false
        this.tokens -= 1
        return true
    }

    private refill() {
        const now = Date.now()
        this.tokens = Math.min(
            CLICK_BURST,
            this.tokens + ((now - this.lastRefillMs) / 1000) * CLICKS_PER_SECOND * this.pace,
        )
        this.lastRefillMs = now
    }

    public listenForBonuses(handlers: BonusHandlers): () => void {
        const identifier = UUIDv4()
        this.bonusCallbacks.set(identifier, handlers)
        handlers.onRules(RULES)
        handlers.onCharges(this.charges)

        return () => this.bonusCallbacks.delete(identifier)
    }

    public async claimBonus(token: string, countryId: string): Promise<ClaimedBonus> {
        const offer = this.offered

        if (!offer || offer.token !== token || budgetNow() > offer.expiresAt) {
            throw new BonusLostError()
        }

        this.offered = undefined
        const claimed = this.grant(offer.reward.kind)

        this.bonusCallbacks.forEach(handlers => handlers.onTaken({countryId}))

        return claimed
    }

    public listenForQuizzes(onOffered: (offer: QuizOffer) => void): () => void {
        const identifier = UUIDv4()
        this.quizCallbacks.set(identifier, onOffered)

        return () => this.quizCallbacks.delete(identifier)
    }

    public async openQuiz(token: string): Promise<QuizQuestion> {
        const quiz = this.quiz
        if (!quiz || quiz.token !== token || budgetNow() > quiz.expiresAt) throw new BonusLostError()

        quiz.deadline ??= budgetNow() + QUIZ_ANSWER_MS

        return {
            text: quiz.asked.text,
            choices: [...quiz.asked.choices],
            deadline: quiz.deadline,
            window: QUIZ_ANSWER_MS,
        }
    }

    public async answerQuiz(token: string, choice: number, countryId: string): Promise<QuizOutcome> {
        const quiz = this.quiz
        if (!quiz || quiz.token !== token || quiz.deadline === undefined) throw new BonusLostError()

        this.quiz = undefined

        const correct = budgetNow() <= quiz.deadline && choice === quiz.asked.correct
        if (!correct) return {correct: false, correctChoice: quiz.asked.correct}

        const kinds = BONUS_KINDS.filter((kind) => !this.full(kind) && (kind !== "bomb" || this.tilePositions))
        if (kinds.length === 0) return {correct: true, correctChoice: quiz.asked.correct}

        const {reward} = this.grant(kinds[Math.floor(Math.random() * kinds.length)])
        this.bonusCallbacks.forEach(handlers => handlers.onTaken({countryId, quizSubject: quiz.asked.subject}))

        return {correct: true, correctChoice: quiz.asked.correct, reward}
    }

    public offerQuiz(): QuizOffer {
        const asked = FAKE_QUIZZES[Math.floor(Math.random() * FAKE_QUIZZES.length)]
        this.quiz = {token: UUIDv4(), expiresAt: budgetNow() + QUIZ_OFFER_TTL_MS, asked}

        const offer: QuizOffer = {token: this.quiz.token, expiresAt: this.quiz.expiresAt}
        this.quizCallbacks.forEach(callback => callback(offer))

        return offer
    }

    public grantBomb(): ClaimedBonus {
        return this.grant("bomb")
    }

    public listenForBombs(onDropped: (drop: BombDrop) => void): () => void {
        const identifier = UUIDv4()
        this.bombCallbacks.set(identifier, onDropped)
        return () => this.bombCallbacks.delete(identifier)
    }

    public async dropBomb(target: GlobePoint, countryId: string): Promise<void> {
        if (this.sessionUnavailable) throw new SessionUnavailableError()
        if (!this.charges.bomb) throw new BonusLostError()

        this.hold({...this.charges, bomb: false})
        await this.explode(target, countryId)
    }

    public async botBomb(tile: number, countryId: string) {
        if (!this.tilePositions) return
        const positions = await this.loadPositions()
        const o = (tile - 1) * 3
        await this.explode({x: positions[o], y: positions[o + 1], z: positions[o + 2]}, countryId)
    }

    private async explode(target: GlobePoint, countryId: string) {
        if (!this.tilePositions) return
        const positions = await this.loadPositions()

        const {tile, arc, point} = nearestTile(positions, target)
        const onLand = tile !== undefined && arc <= SEA_REACH

        const cleared: number[] = []
        const struck: number[] = []
        for (const id of onLand ? tilesWithin(positions, tile, BOMB_RADIUS) : []) {
            const owner = this.tileBindings.get(id)
            if (owner === undefined) continue
            if (this.strike(id)) {
                struck.push(id)
                continue
            }
            this.count(owner, -1)
            this.tileBindings.delete(id)
            cleared.push(id)
        }

        const drop: BombDrop = {
            tile: onLand ? tile : undefined,
            point,
            countryId,
            radius: BOMB_RADIUS,
            cleared,
            struck,
        }
        this.flushUpdates()
        this.bombCallbacks.forEach((callback) => callback(drop))
    }

    private loadPositions(): Promise<Float32Array> {
        this.positions ??= this.tilePositions!()
        return this.positions
    }

    public listenForUpdates(
        callback: (update: Update) => void
    ): () => void {
        const identifier = UUIDv4()
        this.updateListeners.set(identifier, callback)
        return () => {
            this.updateListeners.delete(identifier)
        }
    }

    public listenForUpdatesBatch(
        callback: (updates: Update[]) => void,
    ): () => void {
        const id = UUIDv4()
        this.updateBatchCallbacks.set(id, callback)
        return () => {
            this.updateBatchCallbacks.delete(id)
        }
    }

    public listenForResumes(): () => void {
        return () => {}
    }

    public async getCurrentOwnershipsByBatch(
        batchSize: number,
        maxIndex: number,
        callback: (ownerships: Ownerships) => void,
        signal?: AbortSignal,
    ) {
        for (let start = 1; start <= maxIndex; start += batchSize) {
            signal?.throwIfAborted()

            const bindings = new Map<number, string>()
            const shields = new Map<number, number>()
            const end = Math.min(start + batchSize, maxIndex + 1)
            for (let tile = start; tile < end; tile++) {
                const owner = this.tileBindings.get(tile)
                if (owner) bindings.set(tile, owner)
                const held = this.shields.get(tile)
                if (held) shields.set(tile, held)
            }
            callback({bindings, shields})
        }
    }
}

function drawUpTo(most: number): number {
    return drawBetween(1, most)
}

function drawBetween(least: number, most: number): number {
    return least + Math.floor(Math.random() * (most - least + 1))
}

function rewardOfKind(kind: BonusReward["kind"], amount = 1): BonusReward {
    switch (kind) {
        case "bomb":
            return {kind, radius: BOMB_RADIUS}
        case "encloseClicks":
            return {kind, shapes: amount, maxTiles: ENCLOSE_MAX_TILES}
        case "spreadClicks":
            return {kind, clicks: amount}
        case "refill":
            return {kind}
        case "shields":
            return {kind, shields: amount}
    }
}
