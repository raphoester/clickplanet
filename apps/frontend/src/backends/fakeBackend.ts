import {
    BonusCatch,
    BonusListener,
    BonusLostError,
    BonusOffer,
    Ownerships,
    OwnershipsGetter,
    RateLimitedError,
    TileClicker,
    Update,
    UpdatesListener,
    VPNBlockedError,
} from "./backend.ts";
import {BonusReward, multiplierOf} from "../domain/bonus.ts";
import {ClickBudget, ClickBudgetSource, now as budgetNow} from "./clickBudget.ts";
import {SessionUnavailableError} from "./session.ts";
import {v4 as UUIDv4} from 'uuid';
import {Countries} from "../domain/countries.ts";

const TILE_COUNT = 257_000

const CLICKS_PER_SECOND = 1
const CLICK_BURST = 10

/** Often enough to be worth developing against, not so often it is the game. */
const BONUS_EVERY_MS = 20_000
const BONUS_OFFER_TTL_MS = 15_000
/** The server's defaults: a spread is strong, so it is short and rarer. */
const BONUS_SECONDS: Record<BonusReward["kind"], number> = {tripleClicks: 20, spreadClicks: 10}
const BONUS_KINDS: BonusReward["kind"][] = ["tripleClicks", "tripleClicks", "tripleClicks", "spreadClicks"]

export type FakeBackendOptions = {
    vpnBlocked?: boolean
    sessionUnavailable?: boolean
}

export class FakeBackend implements TileClicker, OwnershipsGetter, UpdatesListener, ClickBudgetSource, BonusListener {
    private tileBindings: Map<number, string> = new Map()
    private updateListeners: Map<string, (update: Update) => void> = new Map()
    private pendingUpdates: Update[] = []
    private updateBatchCallbacks: Map<string, (update: Update[]) => void> = new Map()
    private budgetCallbacks: Map<string, (budget: ClickBudget) => void> = new Map()
    private bonusCallbacks: Map<string, {onOffered: (offer: BonusOffer) => void, onTaken: (taken: BonusCatch) => void}> = new Map()

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

        for (let i = 1; i <= TILE_COUNT; i++) {
            this.tileBindings.set(i, "fr")
        }

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
        this.timers.push(setInterval(() => {
            const offer: BonusOffer = {
                token: UUIDv4(),
                seed: Math.floor(Math.random() * 0xffffffff),
                reward: rewardOfKind(BONUS_KINDS[Math.floor(Math.random() * BONUS_KINDS.length)]),
                expiresAt: budgetNow() + BONUS_OFFER_TTL_MS,
            }

            this.offered = offer
            this.bonusCallbacks.forEach(handlers => handlers.onOffered(offer))
        }, BONUS_EVERY_MS))

        Countries.forEach((country) => {
            let tileId = Math.floor(Math.random() * 10_000)
            const gap = Math.floor(Math.random() * 100)

            this.timers.push(setInterval(() => {
                tileId = (tileId + gap) % TILE_COUNT + 1
                this.applyClick(tileId, country.code)
            }, Math.random() * 1000))
        })
    }

    public close() {
        this.timers.forEach(clearInterval)
        this.timers.length = 0
        clearTimeout(this.bonusEndTimer)
        this.updateListeners.clear()
        this.updateBatchCallbacks.clear()
        this.budgetCallbacks.clear()
    }

    public async clickTile(tileId: number, countryId: string) {
        if (this.sessionUnavailable) throw new SessionUnavailableError()
        if (this.vpnBlocked) throw new VPNBlockedError()

        const allowed = this.allow()
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

    private budget(): ClickBudget {
        this.refill()

        // The policy widens while a bonus runs, exactly as the server's does —
        // so the meter here is driven by the same thing it will be in
        // production rather than by anything the component does itself.
        const boost = this.boost()

        return {
            tokens: this.tokens,
            capacity: CLICK_BURST * boost,
            perSecond: CLICKS_PER_SECOND * boost,
            readAt: budgetNow(),
        }
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
        this.updateListeners.forEach(l => l({
            tile: tileId,
            previousCountry: prev,
            newCountry: countryId,
        }))
    }

    private allow(): boolean {
        this.refill()

        if (this.tokens < 1) return false
        this.tokens -= 1
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
        this.active = offer.reward
        this.activeUntilMs = Date.now() + offer.reward.seconds * 1000
        this.reportBudget()

        // The narrowing is a reading too, or the meter keeps the wide burst.
        clearTimeout(this.bonusEndTimer)
        this.bonusEndTimer = setTimeout(() => this.reportBudget(), offer.reward.seconds * 1000)

        this.bonusCallbacks.forEach(handlers => handlers.onTaken({countryId}))

        return offer.reward
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
    return {kind, seconds: BONUS_SECONDS[kind]}
}
