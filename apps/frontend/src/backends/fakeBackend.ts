import {
    BankFullError,
    BombDrop,
    Bomber,
    BonusHandlers,
    BonusListener,
    BonusLostError,
    BonusOffer,
    ClaimedBonus,
    GlobePoint,
    Enclosure,
    Ownerships,
    OwnershipsGetter,
    QuizMaster,
    RateLimitedError,
    Refiller,
    TileClicker,
    Update,
    UpdatesListener,
    VPNBlockedError,
} from "./backend.ts";
import {ALL_OFF, BonusReward, BonusRules, Charges, NO_CHARGES, Switches} from "../domain/bonus.ts";
import {QuizOffer, QuizOutcome, QuizQuestion} from "../domain/quiz.ts";
import {ClickBudget, ClickBudgetSource, ClickPrice, now as budgetNow} from "./clickBudget.ts";
import {SessionUnavailableError} from "./session.ts";
import {v4 as UUIDv4} from 'uuid';
import {Countries} from "../domain/countries.ts";
import {nearestTile, tilesWithin} from "../domain/blast.ts";

const TILE_COUNT = 257_000

/** Production's `rateLimiter`: one click every 5s, 60 in hand. */
const CLICKS_PER_SECOND = 0.2
const CLICK_BURST = 60

/** Production's `toll.steps`: from each share of the map, the refill is that many times slower. */
const TOLL_STEPS = [
    {share: 0.25, slowdown: 1.5},
    {share: 0.50, slowdown: 2},
    {share: 0.70, slowdown: 3},
]

/** Often enough to be worth developing against, not so often it is the game. */
const BONUS_EVERY_MS = 20_000
const BONUS_OFFER_TTL_MS = 15_000
/**
 * The server's defaults: a pool of 8 spread clicks and a stack of 3 enclosures,
 * a box adding 1 to 4 and 1 to 3 of them, and a shape of 25 tiles at most.
 */
const SPREAD_CLICKS = 8
const SPREAD_PER_BOX = 4
const ENCLOSURES = 3
const ENCLOSURES_PER_BOX = 3
/** The production weights, 5 : 2 : 1 : 2. `giveBomb()` in the console skips the wait. */
const BONUS_KINDS: BonusReward["kind"][] = [
    "refill", "refill", "refill", "refill", "refill",
    "spreadClicks", "spreadClicks",
    "bomb",
    "encloseClicks", "encloseClicks",
]

/** Radians of arc: the server's 8 tile spacings, at its measured spacing of 0.004. */
const BOMB_RADIUS = 0.032

/** How far from the nearest tile an aim still hits land, as the server measures it. */
const SEA_REACH = 0.004

/** Past the nearest tiles and short of the next ring: tiles sit 0.003 to 0.0044 apart. */
const SPREAD_REACH = 0.0052

/** Other players' bombs, so a blast elsewhere on the planet can be watched too. */
// The quiz's own clock, far faster than the server's so the banner can be developed against.
const QUIZ_EVERY_MS = 30_000
const QUIZ_OFFER_TTL_MS = 25_000
const QUIZ_ANSWER_MS = 5_000

/**
 * The fake's whole bank. Three questions, because this is here to develop the banner and the card
 * against, not to be played: the real bank is a thousand generated questions that live on the
 * server and are deliberately never shipped to a browser. See /quiz/README.md.
 */
const FAKE_QUIZZES: {subject: string, text: string, choices: string[], correct: number}[] = [
    {subject: "ee", text: "What is the capital of Estonia?", choices: ["Riga", "Tallinn", "Vilnius"], correct: 1},
    {subject: "np", text: "Which of these shares a border with Nepal?", choices: ["China", "Pakistan", "Myanmar"], correct: 0},
    {subject: "br", text: "How is the capital of Brazil spelled?", choices: ["Brazilia", "Brasilia City", "Brasília"], correct: 2},
]

const BOT_BOMB_EVERY_MS = 25_000

/** Everyone else's clicks, together. */
const BOT_CLICKS_PER_SECOND = 4
const ENCLOSE_MAX_TILES = 25

/** What GetBonusRules answers, from the constants above. */
const RULES: BonusRules = {
    blastRadius: BOMB_RADIUS,
    enclosureMaxTiles: ENCLOSE_MAX_TILES,
    spreadClicks: SPREAD_CLICKS,
    enclosures: ENCLOSURES,
}

export type FakeBackendOptions = {
    vpnBlocked?: boolean
    sessionUnavailable?: boolean
    /**
     * Where the tiles are. The real server picks a bomb's tiles from its own
     * map; this fake has none, so without this it never offers a bomb.
     */
    tilePositions?: () => Promise<Float32Array>
}

export class FakeBackend implements TileClicker, OwnershipsGetter, UpdatesListener, ClickBudgetSource, BonusListener, QuizMaster, Bomber, Refiller {
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

    /** What this player holds, as the server keeps it: one of each kind at most. */
    private charges: Charges = NO_CHARGES
    private positions: Promise<Float32Array> | undefined
    private readonly tilePositions: (() => Promise<Float32Array>) | undefined

    /** The one box outstanding, exactly as the server keeps it. */
    private offered: BonusOffer | undefined

    /**
     * The one quiz outstanding. `deadline` is undefined until it is opened, which is the whole
     * shape of the real thing: the banner is an invitation and the clock is the answer's.
     */
    private quiz: {token: string, expiresAt: number, asked: typeof FAKE_QUIZZES[number], deadline?: number} | undefined

    private readonly timers: ReturnType<typeof setInterval>[] = []
    private tokens = CLICK_BURST
    private lastRefillMs = Date.now()
    /** What the last click's country multiplies the refill by, as the server's bucket keeps it. */
    private pace = 1
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

        this.timers.push(setInterval(() => {
            if (this.pendingUpdates.length === 0) return
            const updates = this.pendingUpdates
            this.pendingUpdates = []
            this.updateBatchCallbacks.forEach(callback => callback(updates))
        }, batchUpdateDurationMs))

        // The real server draws one connected caller and offers the box to them
        // alone. There is only one client here, so it is always this one.
        const offerable = this.tilePositions ? BONUS_KINDS : BONUS_KINDS.filter((kind) => kind !== "bomb")

        this.timers.push(setInterval(() => {
            // A kind another box would add nothing to is not offered.
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
            // As the server does: nothing is asked when there is no charge a right answer could be
            // worth, and only one quiz is outstanding at a time.
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

        // Other players, as one steady trickle. A timer per country at a random
        // period had some firing every few milliseconds, which was thousands of
        // updates a second and enough to make the whole machine lag.
        const runs = codes.map(() => ({tile: Math.floor(Math.random() * TILE_COUNT), gap: 1 + Math.floor(Math.random() * 100)}))
        this.timers.push(setInterval(() => {
            const index = Math.floor(Math.random() * codes.length)
            const run = runs[index]
            run.tile = (run.tile + run.gap) % TILE_COUNT + 1
            this.applyClick(run.tile, codes[index])
        }, 1000 / BOT_CLICKS_PER_SECOND))
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
        // The server refuses both at once, before anything is spent.
        if (switches.spread && switches.enclose) throw new Error("spread and enclose switched on together")
        if (this.sessionUnavailable) throw new SessionUnavailableError()
        if (this.vpnBlocked) throw new VPNBlockedError()

        const allowed = this.allow(countryId)
        this.reportBudget()

        if (!allowed) throw new RateLimitedError()
        this.applyClick(tileId, countryId)
        if (switches.enclose) this.pretendToEnclose(tileId, countryId)
        if (switches.spread) this.announceBonusClick(tileId, countryId)
    }

    /** What the server broadcasts for a click made with spread on. */
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
        }
    }

    /** Keeps what the player holds and tells the stream, as the server does after every change. */
    private hold(charges: Charges) {
        this.charges = charges
        this.bonusCallbacks.forEach(handlers => handlers.onCharges(charges))
    }

    /** Grants a box of `kind`, drawing how much it holds, and answers what was kept, as the server does. */
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
        }
    }

    /** Fills the bank with the refill held, refusing a full bank as the server does. */
    public async useRefill(): Promise<void> {
        if (!this.charges.refill) throw new BonusLostError()

        this.refill()
        if (this.tokens >= CLICK_BURST) throw new BankFullError()

        this.tokens = CLICK_BURST
        this.hold({...this.charges, refill: false})
        this.reportBudget()
    }

    /**
     * A spread click on `tile`, which takes the tiles within about a tile
     * spacing of it. Public for the console: `fakeBackend.botSpread(tile, "fr")`.
     */
    public async botSpread(tile: number, countryId: string) {
        if (!this.tilePositions) return
        const positions = await this.loadPositions()

        const spread = tilesWithin(positions, tile, SPREAD_REACH).filter(id => id !== tile)
        this.applyClick(tile, countryId)
        spread.forEach(id => this.applyClick(id, countryId))
        this.bonusCallbacks.forEach(handlers => handlers.onSpread({countryId, tile, spread}))
    }

    /**
     * Grants a bonus as if a box had just been caught, skipping the box. Only
     * the server's half, like `grantBomb`: see `giveBonus` in main.tsx.
     */
    public grantBonus(kind: Exclude<BonusReward["kind"], "bomb">): ClaimedBonus {
        return this.grant(kind)
    }

    /**
     * This fake has no map geometry, so it cannot find a shape. While enclose
     * is on and an enclosure held, every click pretends it closed one instead: a
     * run of neighbouring ids as the outline, and the ids just past it as the
     * inside. Consecutive ids mostly sit side by side on the globe, so it draws
     * a short streak rather than a shape — enough to develop the effect against.
     */
    private pretendToEnclose(tileId: number, countryId: string) {
        if (this.charges.enclosures === 0) return

        const wall = [0, 1, 2, 3, 4, 5].map(step => tileId + step).filter(id => id <= TILE_COUNT)
        const filled = [6, 7, 8].map(step => tileId + step).filter(id => id <= TILE_COUNT)
        filled.forEach(id => this.applyClick(id, countryId))

        this.hold({...this.charges, enclosures: this.charges.enclosures - 1})

        const enclosure: Enclosure = {countryId, closingTile: tileId, wall, filled, yours: true}
        this.bonusCallbacks.forEach(handlers => handlers.onEnclosed(enclosure))
    }

    /**
     * Stands in for what the real server puts on every click answer, so the
     * counter is live in dev without a backend. There is no latency here, so it
     * is also the one place the counter cannot be caught lying.
     */
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

    public priceFor(countryId: string): void {
        this.budgetCountry = countryId
        this.reportBudget()
    }

    private budget(): ClickBudget {
        this.refill()

        // The refill slows with the last click's country, exactly as the
        // server's does — so the meter here is driven by the same thing it will
        // be in production rather than by the component. The bank never changes
        // size.
        return {
            tokens: this.tokens,
            capacity: CLICK_BURST,
            perSecond: CLICKS_PER_SECOND * this.pace,
            price: this.price(this.budgetCountry),
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

    private applyClick(tileId: number, countryId: string) {
        const prev = this.tileBindings.get(tileId)
        this.tileBindings.set(tileId, countryId)
        this.count(prev, -1)
        this.count(countryId, 1)
        this.updateListeners.forEach(l => l({
            tile: tileId,
            previousCountry: prev,
            newCountry: countryId,
        }))
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
        // What the real client reads at load: the rules, and what is held.
        handlers.onRules(RULES)
        handlers.onCharges(this.charges)

        return () => this.bonusCallbacks.delete(identifier)
    }

    public async claimBonus(token: string, countryId: string): Promise<ClaimedBonus> {
        const offer = this.offered

        // The same four refusals the server has, answered as one: unknown,
        // spent, lapsed, or never this caller's.
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

        // Stamped once: opening twice is the same question with less time on it, never more.
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
        // An answer to a question nobody read is a client that skipped openQuiz.
        if (!quiz || quiz.token !== token || quiz.deadline === undefined) throw new BonusLostError()

        this.quiz = undefined

        const correct = budgetNow() <= quiz.deadline && choice === quiz.asked.correct
        if (!correct) return {correct: false, correctChoice: quiz.asked.correct}

        // The same charges a box pays out in, drawn from the kinds this player is not full on.
        const kinds = BONUS_KINDS.filter((kind) => !this.full(kind) && (kind !== "bomb" || this.tilePositions))
        if (kinds.length === 0) return {correct: true, correctChoice: quiz.asked.correct}

        const {reward} = this.grant(kinds[Math.floor(Math.random() * kinds.length)])
        this.bonusCallbacks.forEach(handlers => handlers.onTaken({countryId, quizSubject: quiz.asked.subject}))

        return {correct: true, correctChoice: quiz.asked.correct, reward}
    }

    /**
     * Puts a quiz up now, skipping the wait. See `giveQuiz` in main.tsx.
     */
    public offerQuiz(): QuizOffer {
        const asked = FAKE_QUIZZES[Math.floor(Math.random() * FAKE_QUIZZES.length)]
        this.quiz = {token: UUIDv4(), expiresAt: budgetNow() + QUIZ_OFFER_TTL_MS, asked}

        const offer: QuizOffer = {token: this.quiz.token, expiresAt: this.quiz.expiresAt}
        this.quizCallbacks.forEach(callback => callback(offer))

        return offer
    }

    /**
     * Grants a bomb as if a box had just been caught, skipping the box. Only
     * the server's half: the globe still has to be told, see `giveBomb` in
     * main.tsx.
     */
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

    /** Someone else's bomb, on `tile`. Public for the console: `fakeBackend.botBomb(tile, "fr")`. */
    public async botBomb(tile: number, countryId: string) {
        if (!this.tilePositions) return
        const positions = await this.loadPositions()
        const o = (tile - 1) * 3
        await this.explode({x: positions[o], y: positions[o + 1], z: positions[o + 2]}, countryId)
    }

    /** Lands a bomb as the server does: on the nearest tile, or in the sea past SEA_REACH. */
    private async explode(target: GlobePoint, countryId: string) {
        if (!this.tilePositions) return
        const positions = await this.loadPositions()

        const {tile, arc, point} = nearestTile(positions, target)
        const onLand = tile !== undefined && arc <= SEA_REACH

        const cleared = onLand ? tilesWithin(positions, tile, BOMB_RADIUS).filter((id) => {
            this.count(this.tileBindings.get(id), -1)
            return this.tileBindings.delete(id)
        }) : []

        const drop: BombDrop = {
            tile: onLand ? tile : undefined,
            point,
            countryId,
            radius: BOMB_RADIUS,
            cleared,
        }
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

    public async getCurrentOwnershipsByBatch(
        batchSize: number,
        maxIndex: number,
        callback: (ownerships: Ownerships) => void,
        signal?: AbortSignal,
    ) {
        for (let start = 1; start <= maxIndex; start += batchSize) {
            signal?.throwIfAborted()

            const bindings = new Map<number, string>()
            const end = Math.min(start + batchSize, maxIndex + 1)
            for (let tile = start; tile < end; tile++) {
                const owner = this.tileBindings.get(tile)
                if (owner) bindings.set(tile, owner)
            }
            callback({bindings})
        }
    }
}

/** From 1 to `most`, evenly. */
function drawUpTo(most: number): number {
    return 1 + Math.floor(Math.random() * most)
}

/** `amount` is what a box added; an offer says only the kind, and reads as one. */
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
    }
}
