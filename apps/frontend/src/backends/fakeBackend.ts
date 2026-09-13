import {
    BombDrop,
    Bomber,
    BonusCatch,
    BonusListener,
    BonusLostError,
    BonusOffer,
    GlobePoint,
    Ownerships,
    OwnershipsGetter,
    RateLimitedError,
    TileClicker,
    Update,
    UpdatesListener,
    VPNBlockedError,
} from "./backend.ts";
import {BonusReward, multiplierOf} from "../domain/bonus.ts";
import {ClickBudget, ClickBudgetSource, ClickPrice, now as budgetNow} from "./clickBudget.ts";
import {SessionUnavailableError} from "./session.ts";
import {v4 as UUIDv4} from 'uuid';
import {Countries} from "../domain/countries.ts";
import {nearestTile, tilesWithin} from "../domain/blast.ts";

const TILE_COUNT = 257_000

const CLICKS_PER_SECOND = 1
const CLICK_BURST = 10

/** Production's `toll.steps`: from each share of the map, a click costs that many tokens. */
const TOLL_STEPS = [
    {share: 0.10, cost: 2},
    {share: 0.25, cost: 3},
    {share: 0.50, cost: 5},
    {share: 0.70, cost: 8},
    {share: 0.90, cost: 10},
]

/** Often enough to be worth developing against, not so often it is the game. */
const BONUS_EVERY_MS = 20_000
const BONUS_OFFER_TTL_MS = 15_000
/** The server's defaults: a spread is strong, so it is short and rarer. */
const BONUS_SECONDS: Record<BonusReward["kind"], number> = {tripleClicks: 20, spreadClicks: 10, bomb: 30}
/** The production weights, 5 : 2 : 1. `giveBomb()` in the console skips the wait. */
const BONUS_KINDS: BonusReward["kind"][] = [
    "tripleClicks", "tripleClicks", "tripleClicks", "tripleClicks", "tripleClicks",
    "spreadClicks", "spreadClicks",
    "bomb",
]

/** Radians of arc: the server's 8 tile spacings, at its measured spacing of 0.004. */
const BOMB_RADIUS = 0.032

/** How far from the nearest tile an aim still hits land, as the server measures it. */
const SEA_REACH = 0.004

/** Other players' bombs, so a blast elsewhere on the planet can be watched too. */
const BOT_BOMB_EVERY_MS = 25_000

/** Everyone else's clicks, together. */
const BOT_CLICKS_PER_SECOND = 4

export type FakeBackendOptions = {
    vpnBlocked?: boolean
    sessionUnavailable?: boolean
    /**
     * Where the tiles are. The real server picks a bomb's tiles from its own
     * map; this fake has none, so without this it never offers a bomb.
     */
    tilePositions?: () => Promise<Float32Array>
}

export class FakeBackend implements TileClicker, OwnershipsGetter, UpdatesListener, ClickBudgetSource, BonusListener, Bomber {
    private tileBindings: Map<number, string> = new Map()
    private tileCounts: Map<string, number> = new Map()
    private budgetCountry = ""
    private updateListeners: Map<string, (update: Update) => void> = new Map()
    private pendingUpdates: Update[] = []
    private updateBatchCallbacks: Map<string, (update: Update[]) => void> = new Map()
    private budgetCallbacks: Map<string, (budget: ClickBudget) => void> = new Map()
    private bonusCallbacks: Map<string, {onOffered: (offer: BonusOffer) => void, onTaken: (taken: BonusCatch) => void}> = new Map()
    private bombCallbacks: Map<string, (drop: BombDrop) => void> = new Map()

    /** When the bomb this client holds stops being droppable; 0 for none. */
    private bombHeldUntilMs = 0
    private positions: Promise<Float32Array> | undefined
    private readonly tilePositions: (() => Promise<Float32Array>) | undefined

    /** The one box outstanding, exactly as the server keeps it. */
    private offered: BonusOffer | undefined

    /** The bonus caught last, and when it runs out. */
    private active: BonusReward | undefined
    private activeUntilMs = 0
    private bonusEndTimer: ReturnType<typeof setTimeout> | undefined
    private readonly timers: ReturnType<typeof setInterval>[] = []
    private tokens = CLICK_BURST
    private lastRefillMs = Date.now()
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
        const kinds = this.tilePositions ? BONUS_KINDS : BONUS_KINDS.filter((kind) => kind !== "bomb")

        this.timers.push(setInterval(() => {
            const offer: BonusOffer = {
                token: UUIDv4(),
                seed: Math.floor(Math.random() * 0xffffffff),
                reward: rewardOfKind(kinds[Math.floor(Math.random() * kinds.length)]),
                expiresAt: budgetNow() + BONUS_OFFER_TTL_MS,
            }

            this.offered = offer
            this.bonusCallbacks.forEach(handlers => handlers.onOffered(offer))
        }, BONUS_EVERY_MS))

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
        clearTimeout(this.bonusEndTimer)
        this.updateListeners.clear()
        this.updateBatchCallbacks.clear()
        this.budgetCallbacks.clear()
        this.bombCallbacks.clear()
    }

    public async clickTile(tileId: number, countryId: string) {
        if (this.sessionUnavailable) throw new SessionUnavailableError()
        if (this.vpnBlocked) throw new VPNBlockedError()

        const allowed = this.allow(countryId)
        this.reportBudget()

        if (!allowed) throw new RateLimitedError()
        this.applyClick(tileId, countryId)
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

        // The policy widens while a bonus runs and narrows with the price,
        // exactly as the server's does — so the meter here is driven by the
        // same thing it will be in production rather than by the component.
        const boost = this.boost()
        const price = this.price(this.budgetCountry)

        return {
            tokens: this.tokens / price.cost,
            capacity: Math.floor(CLICK_BURST * boost / price.cost),
            perSecond: CLICKS_PER_SECOND * boost / price.cost,
            price,
            readAt: budgetNow(),
        }
    }

    private price(countryId: string): ClickPrice {
        const share = (this.tileCounts.get(countryId) ?? 0) / TILE_COUNT

        let cost = 1
        for (const step of TOLL_STEPS) {
            if (share < step.share) return {cost, share, next: step}
            cost = step.cost
        }

        return {cost, share}
    }

    /**
     * A spread multiplies nothing, and this fake has no map geometry to spread
     * with, so it only exercises the announcement and the meter's badge.
     */
    private boost(): number {
        if (!this.active || Date.now() >= this.activeUntilMs) return 1

        return multiplierOf(this.active)
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

        const {cost} = this.price(countryId)
        if (this.tokens < cost) return false
        this.tokens -= cost
        return true
    }

    private refill() {
        const now = Date.now()
        const boost = this.boost()

        this.tokens = Math.min(
            CLICK_BURST * boost,
            this.tokens + ((now - this.lastRefillMs) / 1000) * CLICKS_PER_SECOND * boost,
        )
        this.lastRefillMs = now
    }

    public listenForBonuses(handlers: {
        onOffered: (offer: BonusOffer) => void
        onTaken: (taken: BonusCatch) => void
    }): () => void {
        const identifier = UUIDv4()
        this.bonusCallbacks.set(identifier, handlers)

        return () => this.bonusCallbacks.delete(identifier)
    }

    public async claimBonus(token: string, countryId: string): Promise<BonusReward> {
        const offer = this.offered

        // The same four refusals the server has, answered as one: unknown,
        // spent, lapsed, or never this caller's.
        if (!offer || offer.token !== token || budgetNow() > offer.expiresAt) {
            throw new BonusLostError()
        }

        this.offered = undefined
        if (offer.reward.kind === "bomb") {
            this.bombHeldUntilMs = Date.now() + offer.reward.seconds * 1000
        } else {
            this.active = offer.reward
            this.activeUntilMs = Date.now() + offer.reward.seconds * 1000
            this.reportBudget()
        }

        // The narrowing is a reading too, or the meter keeps the wide burst.
        clearTimeout(this.bonusEndTimer)
        this.bonusEndTimer = setTimeout(() => this.reportBudget(), offer.reward.seconds * 1000)

        this.bonusCallbacks.forEach(handlers => handlers.onTaken({countryId}))

        return offer.reward
    }

    /**
     * Grants a bomb as if a box had just been caught, skipping the box. Only
     * the server's half: the globe still has to be told, see `giveBomb` in
     * main.tsx.
     */
    public grantBomb(): BonusReward {
        const reward = rewardOfKind("bomb")
        this.bombHeldUntilMs = Date.now() + reward.seconds * 1000
        return reward
    }

    public listenForBombs(onDropped: (drop: BombDrop) => void): () => void {
        const identifier = UUIDv4()
        this.bombCallbacks.set(identifier, onDropped)
        return () => this.bombCallbacks.delete(identifier)
    }

    public async dropBomb(target: GlobePoint, countryId: string): Promise<void> {
        if (this.sessionUnavailable) throw new SessionUnavailableError()
        if (Date.now() >= this.bombHeldUntilMs) throw new BonusLostError()

        this.bombHeldUntilMs = 0
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

function rewardOfKind(kind: BonusReward["kind"]): BonusReward {
    if (kind === "bomb") return {kind, seconds: BONUS_SECONDS.bomb, radius: BOMB_RADIUS}
    return {kind, seconds: BONUS_SECONDS[kind]}
}
