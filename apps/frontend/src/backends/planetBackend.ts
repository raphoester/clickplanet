import {
    BankFullError,
    BombDrop,
    Bomber,
    BonusCatch,
    BonusHandlers,
    BonusListener,
    GlobePoint,
    BonusLostError,
    BonusOffer,
    ClaimedBonus,
    ShieldRefusedError,
    Enclosure,
    MapFrozenError,
    Ownerships,
    OwnershipsGetter,
    QuizMaster,
    RateLimitedError,
    Refiller,
    Shielder,
    SpreadClick,
    TileClicker,
    Update,
    UpdatesListener,
    VPNBlockedError,
} from "./backend.ts";
import {ALL_OFF, BonusReward, BonusRules, Charges, NO_CHARGES, Switches} from "../domain/bonus.ts";
import {QuizOffer, QuizOutcome, QuizQuestion} from "../domain/quiz.ts";
import {
    BonusKind,
    ChargesHeld,
    ClickBudget as ClickBudgetMessage,
    GetMapResponse,
    MapFrozen,
    PlanetEvent,
    SharedWith,
} from "../gen/grpc/planet/v1/planet_pb.ts";
import {ClickBudget, ClickBudgetSource, ClickPrice, now as budgetNow, SharedBy} from "./clickBudget.ts";
import {ClickService} from "../gen/grpc/planet/v1/planet_connect.ts";
import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect";
import {createConnectTransport} from "@connectrpc/connect-web";
import {v4 as generateUUID} from 'uuid';
import {Config, NO_TIMEOUT, openStream, retrying} from "./transport.ts";
import {NoSession, SESSION_HEADER, SessionProvider, SessionUnavailableError} from "./session.ts";

export type {Config}

export function newClickServiceClient(config: Config): PromiseClient<typeof ClickService> {
    return createPromiseClient(ClickService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class PlanetBackend implements TileClicker, OwnershipsGetter, UpdatesListener, ClickBudgetSource, BonusListener, QuizMaster, Bomber, Refiller, Shielder {
    private pendingUpdates: Update[] = []
    private readonly updateBatchCallbacks = new Map<string, (updates: Update[]) => void>()
    private readonly updateCallbacks = new Map<string, (update: Update) => void>()
    private readonly bonusCallbacks = new Map<string, BonusHandlers>()
    private readonly quizCallbacks = new Map<string, (offer: QuizOffer) => void>()
    private readonly bombCallbacks = new Map<string, (drop: BombDrop) => void>()
    private readonly budgetCallbacks = new Map<string, (budget: ClickBudget) => void>()
    private readonly resumeCallbacks = new Map<string, () => void>()
    private readonly flushTimer: ReturnType<typeof setInterval>
    private stopListening: () => void

    private streamToken: string | undefined

    private budgetAnchor: ClickBudget | undefined

    private inFlight = 0

    private budgetCountry = ""

    private rules: BonusRules | undefined

    private charges: Charges = NO_CHARGES

    constructor(
        private client: PromiseClient<typeof ClickService>,
        batchUpdateDurationMs: number,
        private session: SessionProvider = new NoSession(),
    ) {
        void this.readBudget()
        void this.readRules()

        this.stopListening = this.openEventStream()
        void this.followIdentity()

        this.listenForUpdates((update) => {
            this.pendingUpdates.push(update)
        })

        this.flushTimer = setInterval(() => this.flushUpdates(), batchUpdateDurationMs)
    }

    private flushUpdates() {
        if (this.pendingUpdates.length === 0) return
        const updates = this.pendingUpdates
        this.pendingUpdates = []
        this.updateBatchCallbacks.forEach(callback => callback(updates))
    }

    public close() {
        clearInterval(this.flushTimer)
        this.stopListening()
        this.updateBatchCallbacks.clear()
        this.updateCallbacks.clear()
        this.bonusCallbacks.clear()
        this.quizCallbacks.clear()
        this.bombCallbacks.clear()
        this.budgetCallbacks.clear()
        this.resumeCallbacks.clear()
        this.pendingUpdates = []
    }

    public async clickTile(tileId: number, countryId: string, switches: Switches = ALL_OFF) {
        this.inFlight++
        this.reportBudget()

        try {
            await this.click(tileId, countryId, switches)
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) {
                throw asClickError(e)
            }

            this.session.invalidate()

            try {
                await this.click(tileId, countryId, switches)
            } catch (retried) {
                throw asClickError(retried)
            }
        } finally {
            this.inFlight--
            this.reportBudget()
        }
    }

    private async click(tileId: number, countryId: string, {spread, enclose}: Switches): Promise<void> {
        const token = await this.session.token()

        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)

        try {
            const res = await retrying(
                () => this.client.click({tileId, countryId, spread, enclose}, {headers}),
                `click ${tileId}`,
            )
            this.anchorBudget(res.budget, countryId)
            this.followSession(token)
            if (spread && this.charges.spreadClicksLeft > 0) {
                this.holdCharges({...this.charges, spreadClicksLeft: this.charges.spreadClicksLeft - 1})
            }
            if (res.gift) {
                void this.readCharges(token)
                void this.readBudget()
            }
        } catch (e) {
            this.anchorBudget(budgetDetailOf(e), countryId)
            throw e
        }
    }

    private async readBudget(): Promise<void> {
        const countryId = this.budgetCountry

        const headers = new Headers()
        const token = await this.session.identity()
        if (token) headers.set(SESSION_HEADER, token)

        try {
            const res = await this.client.getBudget({countryId}, {headers})
            this.anchorBudget(res.budget, countryId)
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unimplemented) return
            console.error("could not read the click budget", e)
        }
    }

    private anchorBudget(budget: ClickBudgetMessage | undefined, countryId: string): void {
        if (!budget || budget.capacity === 0) return
        const priced = this.budgetCountry === "" || countryId === this.budgetCountry

        this.budgetAnchor = {
            tokens: budget.tokens,
            capacity: budget.capacity,
            perSecond: budget.refillPerSecond,
            price: priced ? priceOf(budget) : this.budgetAnchor?.price,
            linkedMultiplier: budget.linkedMultiplier > 1 ? budget.linkedMultiplier : undefined,
            sharedWith: SHARED_BY[budget.sharedWith],
            readAt: budgetNow(),
        }

        this.reportBudget()
    }

    public priceFor(countryId: string): void {
        if (countryId === this.budgetCountry) return

        this.budgetCountry = countryId
        void this.readBudget()
    }

    private reportBudget(): void {
        const anchor = this.budgetAnchor
        if (!anchor) return

        const budget = {...anchor, tokens: anchor.tokens - this.inFlight}
        this.budgetCallbacks.forEach(callback => callback(budget))
    }

    public watchClickBudget(callback: (budget: ClickBudget) => void): () => void {
        const id = generateUUID()
        this.budgetCallbacks.set(id, callback)

        if (this.budgetAnchor) {
            callback({...this.budgetAnchor, tokens: this.budgetAnchor.tokens - this.inFlight})
        }

        return () => this.budgetCallbacks.delete(id)
    }

    public async getCurrentOwnershipsByBatch(
        batchSize: number,
        maxIndex: number,
        callback: (ownerships: Ownerships) => void,
        signal?: AbortSignal,
    ) {
        for (let start = 1; start <= maxIndex; start += batchSize) {
            const endTileId = Math.min(start + batchSize, maxIndex)

            const res = await retrying(
                () => this.client.getMap({startTileId: start, endTileId}, {signal}),
                `getMap ${start}..${endTileId}`,
                signal,
            )

            callback({bindings: bindingsOf(res), shields: shieldsOf(res)})
        }
    }

    private openEventStream(): () => void {
        return openStream(
            (signal) => {
                const token = this.session.heldIdentity()
                this.streamToken = token

                const headers = new Headers()
                if (token) headers.set(SESSION_HEADER, token)

                return this.client.listenForEvents({}, {signal, headers, timeoutMs: NO_TIMEOUT})
            },
            (event) => {
                const update = updateOf(event)
                if (update) {
                    this.updateCallbacks.forEach(callback => callback(update))
                    return
                }

                const offer = offerOf(event)
                if (offer) {
                    this.bonusCallbacks.forEach(handlers => handlers.onOffered(offer))
                    return
                }

                const quiz = quizOf(event)
                if (quiz) {
                    this.quizCallbacks.forEach(callback => callback(quiz))
                    return
                }

                const taken = catchOf(event)
                if (taken) {
                    this.bonusCallbacks.forEach(handlers => handlers.onTaken(taken))
                    return
                }

                const drop = bombOf(event)
                if (drop) {
                    // Earlier tile updates go first, or they repaint over the crater.
                    this.flushUpdates()
                    this.bombCallbacks.forEach(callback => callback(drop))
                    return
                }

                const enclosure = enclosureOf(event)
                if (enclosure) {
                    if (enclosure.yours && this.charges.enclosures > 0) {
                        this.holdCharges({...this.charges, enclosures: this.charges.enclosures - 1})
                    }
                    this.bonusCallbacks.forEach(handlers => handlers.onEnclosed(enclosure))
                    return
                }

                const spread = spreadOf(event)
                if (spread) this.bonusCallbacks.forEach(handlers => handlers.onSpread(spread))
            },
            "planet events",
            {
                onResumed: () => {
                    // Updates from before the gap land before the globe forgets them, or they would outrank the catch-up.
                    this.flushUpdates()
                    this.resumeCallbacks.forEach(callback => callback())
                },
            },
        )
    }

    // A returning player is named from load on, with no Turnstile check.
    private async followIdentity(): Promise<void> {
        const token = await this.session.identity()
        if (token && token !== this.streamToken) {
            this.followSession(token)
            return
        }
        await this.readCharges(token)
    }

    private followSession(token: string | undefined): void {
        if (!token || token === this.streamToken) return

        this.streamToken = token
        this.stopListening()
        this.stopListening = this.openEventStream()

        void this.readCharges(token)
    }

    private async readRules(): Promise<void> {
        try {
            const res = await retrying(() => this.client.getBonusRules({}), "getBonusRules")
            const rules = {
                blastRadius: res.blastRadius,
                enclosureMaxTiles: res.enclosureMaxTiles,
                spreadClicks: res.spreadClicks,
                enclosures: res.enclosures,
                shields: res.shields,
                tileShields: res.tileShields,
                toll: res.tollSteps.map(({share, slowdown}) => ({share, slowdown})),
            }
            this.rules = rules
            this.bonusCallbacks.forEach(handlers => handlers.onRules(rules))
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unimplemented) return
            console.error("could not read the bonus rules", e)
        }
    }

    private async readCharges(token: string | undefined): Promise<void> {
        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)

        try {
            const res = await retrying(() => this.client.getCharges({}, {headers}), "getCharges")
            if (token !== this.streamToken && token !== this.session.heldIdentity()) return
            this.holdCharges(chargesOfMessage(res.charges))
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unimplemented) return
            console.error("could not read the charges held", e)
        }
    }

    private holdCharges(charges: Charges): void {
        this.charges = charges
        this.bonusCallbacks.forEach(handlers => handlers.onCharges(charges))
    }

    public listenForUpdates(callback: (update: Update) => void): () => void {
        const id = generateUUID()
        this.updateCallbacks.set(id, callback)

        return () => this.updateCallbacks.delete(id)
    }

    public listenForResumes(callback: () => void): () => void {
        const id = generateUUID()
        this.resumeCallbacks.set(id, callback)

        return () => this.resumeCallbacks.delete(id)
    }

    public listenForBonuses(handlers: BonusHandlers): () => void {
        const id = generateUUID()
        this.bonusCallbacks.set(id, handlers)

        if (this.rules) handlers.onRules(this.rules)
        handlers.onCharges(this.charges)

        return () => this.bonusCallbacks.delete(id)
    }

    public async claimBonus(token: string, countryId: string): Promise<ClaimedBonus> {
        try {
            return await this.claim(token, countryId)
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) throw asBonusError(e)

            this.session.invalidate()

            try {
                return await this.claim(token, countryId)
            } catch (retried) {
                throw asBonusError(retried)
            }
        }
    }

    private async claim(token: string, countryId: string): Promise<ClaimedBonus> {
        const sessionToken = await this.session.token()

        const headers = new Headers()
        if (sessionToken) headers.set(SESSION_HEADER, sessionToken)

        // No `retrying`: the first request that lands spends the token.
        const res = await this.client.claimBonus({token, countryId}, {headers})
        this.followSession(sessionToken)

        const reward = rewardOf(res.kind, this.rules, res.amount)
        if (!reward) throw new BonusLostError()

        const charges = chargesOfMessage(res.charges)
        this.holdCharges(charges)

        return {reward, charges}
    }

    public listenForQuizzes(onOffered: (offer: QuizOffer) => void): () => void {
        const id = generateUUID()
        this.quizCallbacks.set(id, onOffered)

        return () => this.quizCallbacks.delete(id)
    }

    public async openQuiz(token: string): Promise<QuizQuestion> {
        try {
            return await this.open(token)
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) throw asBonusError(e)

            this.session.invalidate()

            try {
                return await this.open(token)
            } catch (retried) {
                throw asBonusError(retried)
            }
        }
    }

    private async open(token: string): Promise<QuizQuestion> {
        const sessionToken = await this.session.token()

        const headers = new Headers()
        if (sessionToken) headers.set(SESSION_HEADER, sessionToken)

        const at = budgetNow()
        const wallClock = Date.now()
        const res = await this.client.openQuiz({token}, {headers})
        this.followSession(sessionToken)

        return {
            text: res.question,
            choices: [...res.choices],
            deadline: at + (Number(res.deadlineUnixMs) - wallClock),
            window: res.answerSeconds * 1000,
        }
    }

    public async answerQuiz(token: string, choice: number, countryId: string): Promise<QuizOutcome> {
        const sessionToken = await this.session.token()

        const headers = new Headers()
        if (sessionToken) headers.set(SESSION_HEADER, sessionToken)

        let res
        try {
            res = await this.client.answerQuiz({token, choice, countryId}, {headers})
        } catch (e) {
            throw asBonusError(e)
        }
        this.followSession(sessionToken)

        this.holdCharges(chargesOfMessage(res.charges))

        return {
            correct: res.correct,
            correctChoice: res.correctChoice,
            reward: res.correct ? rewardOf(res.kind, this.rules, res.amount) : undefined,
        }
    }

    public listenForBombs(onDropped: (drop: BombDrop) => void): () => void {
        const id = generateUUID()
        this.bombCallbacks.set(id, onDropped)

        return () => this.bombCallbacks.delete(id)
    }

    public async dropBomb(target: GlobePoint, countryId: string): Promise<void> {
        this.holdCharges({...this.charges, bomb: false})

        try {
            await this.dropRetried(target, countryId)
        } catch (e) {
            if (!(e instanceof BonusLostError) && !this.charges.bomb) this.holdCharges({...this.charges, bomb: true})
            throw e
        }
    }

    private async dropRetried(target: GlobePoint, countryId: string): Promise<void> {
        try {
            await this.drop(target, countryId)
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) throw asBonusError(e)

            this.session.invalidate()

            try {
                await this.drop(target, countryId)
            } catch (retried) {
                throw asBonusError(retried)
            }
        }
    }

    private async drop(target: GlobePoint, countryId: string): Promise<void> {
        const sessionToken = await this.session.token()

        const headers = new Headers()
        if (sessionToken) headers.set(SESSION_HEADER, sessionToken)

        await this.client.dropBomb({target, countryId}, {headers})
        this.followSession(sessionToken)
    }

    public async useRefill(countryId: string): Promise<void> {
        try {
            await this.refill(countryId)
        } catch (e) {
            let error = asRefillError(e)
            if (e instanceof ConnectError && e.code === Code.Unauthenticated) {
                this.session.invalidate()
                try {
                    await this.refill(countryId)
                    return
                } catch (retried) {
                    error = asRefillError(retried)
                }
            }
            if (error instanceof BonusLostError) this.holdCharges({...this.charges, refill: false})
            throw error
        }
    }

    private async refill(countryId: string): Promise<void> {
        const sessionToken = await this.session.token()

        const headers = new Headers()
        if (sessionToken) headers.set(SESSION_HEADER, sessionToken)

        const res = await this.client.useRefill({countryId}, {headers})
        this.followSession(sessionToken)
        this.anchorBudget(res.budget, countryId)
        this.holdCharges(chargesOfMessage(res.charges))
    }

    public async placeShield(tileId: number, countryId: string): Promise<void> {
        this.holdCharges({...this.charges, shields: Math.max(0, this.charges.shields - 1)})

        try {
            await this.placeRetried(tileId, countryId)
        } catch (e) {
            this.holdCharges({...this.charges, shields: e instanceof BonusLostError ? 0 : this.charges.shields + 1})
            throw e
        }
    }

    private async placeRetried(tileId: number, countryId: string): Promise<void> {
        try {
            await this.place(tileId, countryId)
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) throw asShieldError(e)

            this.session.invalidate()

            try {
                await this.place(tileId, countryId)
            } catch (retried) {
                throw asShieldError(retried)
            }
        }
    }

    private async place(tileId: number, countryId: string): Promise<void> {
        const sessionToken = await this.session.token()

        const headers = new Headers()
        if (sessionToken) headers.set(SESSION_HEADER, sessionToken)

        const res = await this.client.placeShield({tileId, countryId}, {headers})
        this.followSession(sessionToken)
        this.holdCharges(chargesOfMessage(res.charges))
    }

    public listenForUpdatesBatch(
        callback: (updates: Update[]) => void,
    ): () => void {
        const id = generateUUID()
        this.updateBatchCallbacks.set(id, callback)
        return () => this.updateBatchCallbacks.delete(id)
    }
}

export function offerOf(event: PlanetEvent, at = budgetNow(), wallClock = Date.now()): BonusOffer | undefined {
    if (event.event.case !== "bonusOffered") return undefined

    const offered = event.event.value

    const reward = rewardOf(offered.kind)
    if (!reward) return undefined

    return {
        token: offered.token,
        seed: offered.seed,
        reward,
        expiresAt: at + (Number(offered.expiresAtUnixMs) - wallClock),
    }
}

export function quizOf(event: PlanetEvent, at = budgetNow(), wallClock = Date.now()): QuizOffer | undefined {
    if (event.event.case !== "quizOffered") return undefined

    const offered = event.event.value

    return {
        token: offered.token,
        expiresAt: at + (Number(offered.expiresAtUnixMs) - wallClock),
    }
}

export function catchOf(event: PlanetEvent): BonusCatch | undefined {
    if (event.event.case !== "bonusTaken") return undefined

    const taken = event.event.value
    return {countryId: taken.countryId, quizSubject: taken.quizSubjectCountryId || undefined}
}

export function enclosureOf(event: PlanetEvent): Enclosure | undefined {
    if (event.event.case !== "tilesEnclosed") return undefined

    const enclosed = event.event.value
    return {
        countryId: enclosed.countryId,
        closingTile: enclosed.closingTileId,
        wall: [...enclosed.wallTileIds],
        filled: [...enclosed.filledTileIds],
        yours: enclosed.yours || undefined,
    }
}

export function spreadOf(event: PlanetEvent): SpreadClick | undefined {
    if (event.event.case !== "tilesSpread") return undefined

    const spread = event.event.value
    return {countryId: spread.countryId, tile: spread.tileId, spread: [...spread.spreadTileIds]}
}

export function chargesOfMessage(held: ChargesHeld | undefined): Charges {
    if (!held) return NO_CHARGES

    return {
        refill: held.refill,
        bomb: held.bomb,
        enclosures: held.enclosures,
        spreadClicksLeft: held.spreadClicksLeft,
        shields: held.shields,
    }
}

const NO_RULES: BonusRules = {blastRadius: 0, enclosureMaxTiles: 0, spreadClicks: 0, enclosures: 0, shields: 0, tileShields: 0, toll: []}

function rewardOf(
    kind: BonusKind,
    {blastRadius, enclosureMaxTiles: maxTiles}: BonusRules = NO_RULES,
    amount = 1,
): BonusReward | undefined {
    switch (kind) {
        case BonusKind.REFILL:
            return {kind: "refill"}
        case BonusKind.SPREAD_CLICKS:
            return {kind: "spreadClicks", clicks: amount}
        case BonusKind.BOMB:
            return {kind: "bomb", radius: blastRadius}
        case BonusKind.ENCLOSE_CLICKS:
            return {kind: "encloseClicks", shapes: amount, maxTiles}
        case BonusKind.SHIELDS:
            return {kind: "shields", shields: amount}
        default:
            return undefined
    }
}

export function bombOf(event: PlanetEvent): BombDrop | undefined {
    if (event.event.case !== "bombDropped") return undefined

    const dropped = event.event.value
    return {
        tile: dropped.tileId === 0 ? undefined : dropped.tileId,
        point: {x: dropped.point?.x ?? 0, y: dropped.point?.y ?? 0, z: dropped.point?.z ?? 1},
        countryId: dropped.countryId,
        radius: dropped.radius,
        cleared: dropped.clearedTileIds,
        struck: dropped.struckTileIds,
    }
}

export function asBonusError(e: unknown): unknown {
    if (e instanceof SessionUnavailableError) return e

    if (e instanceof ConnectError) {
        if (frozen(e)) return new MapFrozenError({cause: e})
        if (e.code === Code.NotFound || e.code === Code.Unimplemented) return new BonusLostError({cause: e})
        if (e.code === Code.Unauthenticated) return new SessionUnavailableError({cause: e})
    }

    return e
}

export function asShieldError(e: unknown): unknown {
    if (e instanceof ConnectError && e.code === Code.FailedPrecondition && !frozen(e)) return new ShieldRefusedError({cause: e})

    return asBonusError(e)
}

export function asRefillError(e: unknown): unknown {
    if (e instanceof ConnectError && e.code === Code.FailedPrecondition && !frozen(e)) return new BankFullError({cause: e})

    return asBonusError(e)
}

function frozen(e: ConnectError): boolean {
    return e.code === Code.FailedPrecondition && e.findDetails(MapFrozen).length > 0
}

const SHARED_BY: Partial<Record<SharedWith, SharedBy>> = {
    [SharedWith.GUESTS]: "guests",
    [SharedWith.NETWORK]: "network",
}

export function priceOf(budget: ClickBudgetMessage): ClickPrice | undefined {
    if (budget.slowdown === 0) return undefined

    return {
        slowdown: budget.slowdown,
        share: budget.share,
        next: budget.nextSlowdown === 0 ? undefined : {share: budget.nextShare, slowdown: budget.nextSlowdown},
    }
}

export function budgetDetailOf(e: unknown): ClickBudgetMessage | undefined {
    if (!(e instanceof ConnectError)) return undefined

    return e.findDetails(ClickBudgetMessage)[0]
}

export function asClickError(e: unknown): unknown {
    if (e instanceof SessionUnavailableError) return e

    if (e instanceof ConnectError) {
        if (frozen(e)) return new MapFrozenError({cause: e})
        if (e.code === Code.ResourceExhausted) return new RateLimitedError({cause: e})
        if (e.code === Code.PermissionDenied) return new VPNBlockedError({cause: e})
        if (e.code === Code.Unauthenticated) return new SessionUnavailableError({cause: e})
    }

    return e
}

export function bindingsOf(res: GetMapResponse): Map<number, string> {
    const tiles = new DataView(res.tiles.buffer, res.tiles.byteOffset, res.tiles.byteLength)

    const bindings = new Map<number, string>()
    for (let offset = 0; offset + 1 < res.tiles.byteLength; offset += 2) {
        const code = tiles.getUint16(offset, true)
        if (code === 0) continue
        bindings.set(res.startTileId + offset / 2, res.codes[code])
    }

    return bindings
}

export function shieldsOf(res: GetMapResponse): Map<number, number> {
    return new Map(res.shields.map(({tileId, shields}) => [tileId, shields]))
}

export function updateOf(event: PlanetEvent): Update | undefined {
    if (event.event.case !== "tileUpdate") return undefined

    const update = event.event.value
    return {
        tile: update.tileId,
        previousCountry: update.previousCountryId === "" ? undefined : update.previousCountryId,
        newCountry: update.countryId === "" ? undefined : update.countryId,
        clicked: update.clicked,
        shields: update.shields,
    }
}
