import {
    AllegianceLockedError,
    AllegianceSource,
    BankFullError,
    BannedError,
    BombDrop,
    Bomber,
    BonusHandlers,
    BonusListener,
    BombRefusedError,
    BonusLostError,
    ChargesRefresher,
    ClaimedBonus,
    Counts,
    Enclosure,
    Fortification,
    KnockOut,
    GlobePoint,
    Ownerships,
    OwnershipsGetter,
    QuizMaster,
    RateLimitedError,
    Refiller,
    Shielder,
    ShieldRefusedError,
    SpreadClick,
    TileClicker,
    LandmassReader,
    LandmassState,
    MapVersion,
    TileHolder,
    TileHolders,
    TileRange,
    Update,
    UpdatesListener,
    VPNBlockedError,
    FrozenError,
    OutOfReachError,
} from "./backend.ts";
import {ALL_OFF, BonusRules, Charges, NO_CHARGES, Switches} from "../domain/bonus.ts";
import {QuizOffer, QuizOutcome, QuizQuestion} from "../domain/quiz.ts";
import {Allegiance} from "../domain/allegiance.ts";
import {
    Allegiance as AllegianceMessage,
    BanRefusal,
    BombRefusal,
    FrozenRefusal,
    ChargesHeld,
    ClickBudget as ClickBudgetMessage,
    GetHoldingsResponse,
    GetMapResponse,
    HoldingsChanged,
    LandmassHoldings,
    MapVersion as MapVersionMessage,
    PlanetEvent,
    ReachRefusal,
    SharedWith,
} from "../gen/grpc/planet/v1/planet_pb.ts";
import {ClickBudget, ClickBudgetSource, ClickPrice, now as budgetNow, SharedBy} from "./clickBudget.ts";
import {ClickService} from "../gen/grpc/planet/v1/planet_connect.ts";
import {BonusService} from "../gen/grpc/bonuses/v1/bonuses_connect.ts";
import {catchOf, grantRulesChanged, offerOf, quizOf} from "./bonusesBackend.ts";
import {rewardOf} from "./bonusKind.ts";
import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect";
import {createConnectTransport} from "@connectrpc/connect-web";
import {v4 as generateUUID} from 'uuid';
import {Config, NO_TIMEOUT, openStream, retrying, retryingAtOnce} from "./transport.ts";
import {NoSession, SESSION_HEADER, SessionProvider, SessionUnavailableError} from "./session.ts";

export type {Config}

export function newClickServiceClient(config: Config): PromiseClient<typeof ClickService> {
    return createPromiseClient(ClickService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class PlanetBackend implements TileClicker, OwnershipsGetter, UpdatesListener, ClickBudgetSource, BonusListener, QuizMaster, Bomber, Refiller, Shielder, AllegianceSource, ChargesRefresher, TileHolders, LandmassReader {
    private pendingUpdates: Update[] = []
    private readonly updateBatchCallbacks = new Map<string, (updates: Update[]) => void>()
    private readonly updateCallbacks = new Map<string, (update: Update) => void>()
    private readonly bonusCallbacks = new Map<string, BonusHandlers>()
    private readonly quizCallbacks = new Map<string, (offer: QuizOffer) => void>()
    private readonly bombCallbacks = new Map<string, (drop: BombDrop) => void>()
    private readonly budgetCallbacks = new Map<string, (budget: ClickBudget) => void>()
    private readonly openingCallbacks = new Map<string, (version: MapVersion) => void>()
    private readonly holdingsCallbacks = new Map<string, (change: Counts) => void>()
    private readonly fortificationCallbacks = new Map<string, (fortification: Fortification) => void>()
    private readonly knockOutCallbacks = new Map<string, (knockOut: KnockOut) => void>()
    private readonly allegianceCallbacks = new Map<string, (allegiance: Allegiance | undefined) => void>()
    private readonly flushTimer: ReturnType<typeof setInterval>
    private stopListening: () => void
    private stopOffers: () => void

    private streamToken: string | undefined
    private offersToken: string | undefined

    // The version of the last thing the stream told its listeners, and whether the next message opens a connection.
    private streamVersion: MapVersion | undefined
    private opening = false

    private budgetAnchor: ClickBudget | undefined

    private inFlight = 0

    private budgetCountry = ""

    private rules: BonusRules | undefined

    // The offers' rule, read apart from planet's and told inside BonusRules.
    private signInRefill = false

    private charges: Charges = NO_CHARGES

    private allegiance: Allegiance | undefined

    private flagged: {token: string, ready: Promise<void>} | undefined

    // With no offers client, as in a test of the map alone, no box or quiz is offered and none is claimed.
    constructor(
        private client: PromiseClient<typeof ClickService>,
        batchUpdateDurationMs: number,
        private session: SessionProvider = new NoSession(),
        private offers?: PromiseClient<typeof BonusService>,
    ) {
        void this.readBudget()
        void this.readRules()
        void this.readGrantRules()

        this.stopListening = this.openEventStream()
        this.stopOffers = this.openOffersStream()
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
        this.stopOffers()
        this.updateBatchCallbacks.clear()
        this.updateCallbacks.clear()
        this.bonusCallbacks.clear()
        this.quizCallbacks.clear()
        this.bombCallbacks.clear()
        this.budgetCallbacks.clear()
        this.openingCallbacks.clear()
        this.holdingsCallbacks.clear()
        this.fortificationCallbacks.clear()
        this.knockOutCallbacks.clear()
        this.allegianceCallbacks.clear()
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
        await this.ensureFlag(token, countryId)

        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)

        try {
            const res = await retryingAtOnce(
                () => this.client.click({tileId, countryId, spread, enclose}, {headers}),
                `click ${tileId}`,
            )
            this.anchorBudget(res.budget, countryId)
            this.followSession(token)
            if (spread && !this.rules?.spreadUnlimited && this.charges.spreadClicksLeft > 0) {
                this.holdCharges({...this.charges, spreadClicksLeft: this.charges.spreadClicksLeft - 1})
            }
        } catch (e) {
            this.anchorBudget(budgetDetailOf(e), countryId)
            this.holdRefusal(e)
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
            frozen: budget.frozen,
            fastStart: budget.fastStart,
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

    public async getOwnerships(
        ranges: readonly TileRange[],
        callback: (ownerships: Ownerships) => void,
        signal?: AbortSignal,
    ) {
        const reading = new AbortController()
        const stop = () => reading.abort(signal?.reason)
        if (signal?.aborted) stop()
        signal?.addEventListener("abort", stop, {once: true})

        const batches = [...ranges]
        const read = async () => {
            for (let range = batches.shift(); range; range = batches.shift()) {
                const {start, end} = range
                const res = await retrying(
                    () => this.client.getMap({startTileId: start, endTileId: end}, {signal: reading.signal}),
                    `getMap ${start}..${end}`,
                    reading.signal,
                )
                reading.signal.throwIfAborted()
                callback(ownershipsOf(res, range))
            }
        }

        await Promise.all(Array.from({length: MAP_READS_IN_FLIGHT}, () => read().catch((e) => reading.abort(e))))
        signal?.removeEventListener("abort", stop)
        reading.signal.throwIfAborted()
    }

    public async getHoldings(signal?: AbortSignal): Promise<Counts> {
        return countsOf(await retrying(() => this.client.getHoldings({}, {signal}), "getHoldings", signal))
    }

    public async getFortresses(signal?: AbortSignal): Promise<Map<number, string>> {
        try {
            const res = await retrying(() => this.client.getFortresses({}, {signal}), "getFortresses", signal)
            return new Map(res.fortresses.map(({landmassId, countryId}) => [landmassId, countryId]))
        } catch (e) {
            // A server from before fortifying has no locks to tell.
            if (e instanceof ConnectError && e.code === Code.Unimplemented) return new Map()
            throw e
        }
    }

    public async tileHolder(tile: number, signal?: AbortSignal): Promise<TileHolder | undefined> {
        const {holder} = await this.client.getTileHolder({tileId: tile}, {signal})
        if (!holder) return undefined
        return {accountId: holder.accountId, countryCode: holder.countryId, takenAt: Number(holder.takenAtUnixMs)}
    }

    public async landmass(landmass: number, signal?: AbortSignal): Promise<LandmassState> {
        const {holders, lastFortification: last} = await this.client.getLandmass({landmassId: landmass}, {signal})
        return {
            holders: holders.map(({accountId, countryId, tiles}) => ({accountId, countryCode: countryId, tiles})),
            lastFortification: last && {accountId: last.accountId, countryCode: last.countryId, at: Number(last.fortifiedAtUnixMs)},
        }
    }

    private openEventStream(): () => void {
        return openStream(
            (signal) => {
                const token = this.session.heldIdentity()
                this.streamToken = token

                const headers = new Headers()
                if (token) headers.set(SESSION_HEADER, token)

                this.opening = true
                return this.client.listenForEvents({}, {signal, headers, timeoutMs: NO_TIMEOUT})
            },
            (event) => {
                const opened = this.opening
                this.opening = false

                const heartbeat = heartbeatOf(event)
                if (heartbeat) {
                    if (opened) this.streamOpened(heartbeat)
                    return
                }

                const update = updateOf(event)
                if (update) {
                    this.streamVersion = update.version
                    this.updateCallbacks.forEach(callback => callback(update))
                    return
                }

                const holdings = holdingsChangeOf(event)
                if (holdings) {
                    this.flushUpdates()
                    this.streamVersion = holdings.version
                    this.holdingsCallbacks.forEach(callback => callback(holdings))
                    return
                }

                const drop = bombOf(event)
                if (drop) {
                    // Earlier tile updates go first, or they repaint over the crater.
                    this.flushUpdates()
                    this.streamVersion = drop.version
                    this.bombCallbacks.forEach(callback => callback(drop))
                    return
                }

                if (rulesChanged(event)) {
                    this.readRulesInForce()
                    return
                }

                const fortification = fortificationOf(event)
                if (fortification) {
                    // Its shields go on the owners the earlier updates leave.
                    this.flushUpdates()
                    this.streamVersion = fortification.version
                    this.fortificationCallbacks.forEach(callback => callback(fortification))
                    return
                }

                const knockOut = knockOutOf(event)
                if (knockOut) {
                    // The take of the capital goes first.
                    this.flushUpdates()
                    this.streamVersion = knockOut.version
                    this.knockOutCallbacks.forEach(callback => callback(knockOut))
                    return
                }

                const enclosure = enclosureOf(event)
                if (enclosure) {
                    // Earlier tile updates go first, or they repaint over the tiles it took.
                    this.flushUpdates()
                    this.streamVersion = enclosure.version
                    if (enclosure.yours && !this.rules?.enclosuresUnlimited && this.charges.enclosures > 0) {
                        this.holdCharges({...this.charges, enclosures: this.charges.enclosures - 1})
                    }
                    this.bonusCallbacks.forEach(handlers => handlers.onEnclosed(enclosure))
                    return
                }

                const granted = grantedOf(event)
                if (granted) {
                    this.holdCharges(granted)
                    return
                }

                const spread = spreadOf(event)
                if (spread) this.bonusCallbacks.forEach(handlers => handlers.onSpread(spread))
            },
            "planet events",
            // A gap in the stream may have hidden a rulesChanged or a grant.
            {onResumed: () => {
                this.readRulesInForce()
                void this.refreshCharges()
            }},
        )
    }

    // Opened as the planet's is, and beside it: the boxes and quizzes go only to a caller with it open.
    private openOffersStream(): () => void {
        const offers = this.offers
        if (!offers) return () => {}

        return openStream(
            (signal) => {
                const token = this.session.heldIdentity()
                this.offersToken = token

                const headers = new Headers()
                if (token) headers.set(SESSION_HEADER, token)

                return offers.listenForEvents({}, {signal, headers, timeoutMs: NO_TIMEOUT})
            },
            (event) => {
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

                if (grantRulesChanged(event)) void this.readGrantRules()
            },
            "bonus events",
            // A gap in the stream may have hidden a rulesChanged.
            {onResumed: () => void this.readGrantRules()},
        )
    }

    private readRulesInForce(): void {
        void this.readBudget()
        void this.readRules()
        const token = this.streamToken ?? this.session.heldIdentity()
        if (token) void this.readAllegiance(token)
    }

    // Updates from before a gap land before the globe hears of it, or they would outrank what it reads again.
    private streamOpened(version: MapVersion) {
        this.flushUpdates()
        this.streamVersion = version
        this.openingCallbacks.forEach(callback => callback(version))
    }

    // A returning player is named from load on, with no Turnstile check.
    private async followIdentity(): Promise<void> {
        const token = await this.session.identity()
        if (token && token !== this.streamToken) {
            this.followSession(token)
            return
        }
        this.followOffers(token)
        await Promise.all([this.readCharges(token), this.readAllegiance(token)])
    }

    private followSession(token: string | undefined): void {
        this.followOffers(token)
        if (!token || token === this.streamToken) return

        this.streamToken = token
        this.stopListening()
        this.stopListening = this.openEventStream()

        void this.readCharges(token)
        void this.readAllegiance(token)
    }

    // The offers stream reads the token only when it opens, as the planet's does, and each keeps its own: either can
    // reconnect alone.
    private followOffers(token: string | undefined): void {
        if (!token || token === this.offersToken) return

        this.offersToken = token
        this.stopOffers()
        this.stopOffers = this.openOffersStream()
    }

    private async readRules(): Promise<void> {
        try {
            const res = await retrying(() => this.client.getChargeRules({}), "getChargeRules")
            const rules: BonusRules = {
                blastRadius: res.blastRadius,
                enclosureMaxTiles: res.enclosureMaxTiles,
                spreadClicks: res.spreadClicks,
                enclosures: res.enclosures,
                shields: res.shields,
                tileShields: res.tileShields,
                toll: res.tollSteps.map(({share, slowdown}) => ({share, slowdown})),
                signInRefill: this.signInRefill,
                enclosuresUnlimited: res.enclosuresUnlimited,
                reach: res.reach,
                knockOut: res.knockOut,
                capitalShieldsFixed: res.capitalShieldsFixed,
                bombsOff: res.bombsOff,
                enclosuresOff: res.enclosuresOff,
                spreadUnlimited: res.spreadUnlimited,
            }
            this.holdRules(rules)
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unimplemented) return
            console.error("could not read the charge rules", e)
        }
    }

    // Told with planet's rules: until those are read, there are no rules to tell.
    private async readGrantRules(): Promise<void> {
        const offers = this.offers
        if (!offers) return

        try {
            const res = await retrying(() => offers.getGrantRules({}), "getGrantRules")
            this.signInRefill = res.signInRefill
            if (this.rules && this.rules.signInRefill !== res.signInRefill) {
                this.holdRules({...this.rules, signInRefill: res.signInRefill})
            }
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unimplemented) return
            console.error("could not read the grant rules", e)
        }
    }

    private holdRules(rules: BonusRules): void {
        this.rules = rules
        this.bonusCallbacks.forEach(handlers => handlers.onRules(rules))
    }

    public async refreshCharges(): Promise<void> {
        await this.readCharges(this.streamToken ?? this.session.heldIdentity())
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

    private async readAllegiance(token: string | undefined): Promise<void> {
        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)

        try {
            const res = await retrying(() => this.client.getAllegiance({}, {headers}), "getAllegiance")
            if (token !== this.streamToken && token !== this.session.heldIdentity()) return
            const held = allegianceOfMessage(res.allegiance)
            this.holdAllegiance(held)
            if (held && token) this.flagged = {token, ready: Promise.resolve()}
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unimplemented) return
            console.error("could not read the flag played for", e)
        }
    }

    public async refreshAllegiance(): Promise<void> {
        await this.readAllegiance(this.streamToken ?? this.session.heldIdentity())
    }

    public async pledgeAllegiance(countryId: string): Promise<void> {
        const token = await this.session.identity()
        if (token) await this.pledge(token, countryId)
    }

    // The page's flag is the one its clicks are priced for.
    public async claimFlag(): Promise<void> {
        if (this.budgetCountry === "") return
        await this.ensureFlag(await this.session.token(), this.budgetCountry)
    }

    private ensureFlag(token: string | undefined, countryId: string): Promise<void> {
        if (!token) return Promise.resolve()
        if (this.flagged?.token === token) return this.flagged.ready

        const flagged = {token, ready: this.claimFlagFor(token, countryId)}
        this.flagged = flagged
        flagged.ready.catch(() => {
            if (this.flagged === flagged) this.flagged = undefined
        })
        return flagged.ready
    }

    private async claimFlagFor(token: string, countryId: string): Promise<void> {
        if (await this.heldFor(token) === undefined) await this.pledge(token, countryId)
    }

    private async heldFor(token: string): Promise<Allegiance | undefined> {
        const headers = new Headers({[SESSION_HEADER]: token})
        const res = await retrying(() => this.client.getAllegiance({}, {headers}), "getAllegiance")
        const held = allegianceOfMessage(res.allegiance)
        this.holdAllegiance(held)
        return held
    }

    private async pledge(token: string, countryId: string): Promise<void> {
        const headers = new Headers({[SESSION_HEADER]: token})
        try {
            await retrying(() => this.client.pledgeAllegiance({countryId}, {headers}), "pledgeAllegiance")
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unimplemented) return
            const locked = allegianceDetailOf(e)
            if (!locked) throw e
            this.holdAllegiance(locked)
            throw new AllegianceLockedError(locked, {cause: e})
        }

        await this.heldFor(token)
        this.flagged = {token, ready: Promise.resolve()}
    }

    private holdRefusal(e: unknown): void {
        const locked = allegianceDetailOf(e)
        if (locked) this.holdAllegiance(locked)
    }

    private holdAllegiance(allegiance: Allegiance | undefined): void {
        this.allegiance = allegiance
        this.allegianceCallbacks.forEach(callback => callback(allegiance))
    }

    public listenForAllegiance(callback: (allegiance: Allegiance | undefined) => void): () => void {
        const id = generateUUID()
        this.allegianceCallbacks.set(id, callback)
        callback(this.allegiance)

        return () => this.allegianceCallbacks.delete(id)
    }

    public listenForUpdates(callback: (update: Update) => void): () => void {
        const id = generateUUID()
        this.updateCallbacks.set(id, callback)

        return () => this.updateCallbacks.delete(id)
    }

    public listenForOpenings(callback: (version: MapVersion) => void): () => void {
        const id = generateUUID()
        this.openingCallbacks.set(id, callback)

        // What it missed is older than this, and a read from now on holds it.
        this.flushUpdates()
        if (this.streamVersion) callback(this.streamVersion)

        return () => this.openingCallbacks.delete(id)
    }

    public listenForHoldings(callback: (change: Counts) => void): () => void {
        const id = generateUUID()
        this.holdingsCallbacks.set(id, callback)
        return () => this.holdingsCallbacks.delete(id)
    }

    public listenForFortifications(callback: (fortification: Fortification) => void): () => void {
        const id = generateUUID()
        this.fortificationCallbacks.set(id, callback)

        return () => this.fortificationCallbacks.delete(id)
    }

    public listenForKnockOuts(callback: (knockOut: KnockOut) => void): () => void {
        const id = generateUUID()
        this.knockOutCallbacks.set(id, callback)

        return () => this.knockOutCallbacks.delete(id)
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
        const res = await this.offersClient().claimBonus({token, countryId}, {headers})
        this.followSession(sessionToken)

        const reward = rewardOf(res.kind, this.rules, res.amount)
        if (!reward) throw new BonusLostError()

        const charges = chargesOfMessage(res.charges)
        this.holdCharges(charges)

        return {reward, charges}
    }

    private offersClient(): PromiseClient<typeof BonusService> {
        if (!this.offers) throw new BonusLostError()
        return this.offers
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
        const res = await this.offersClient().openQuiz({token}, {headers})
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
            res = await this.offersClient().answerQuiz({token, choice, countryId}, {headers})
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
        await this.ensureFlag(sessionToken, countryId)

        const headers = new Headers()
        if (sessionToken) headers.set(SESSION_HEADER, sessionToken)

        try {
            const res = await this.client.placeShield({tileId, countryId}, {headers})
            this.followSession(sessionToken)
            this.holdCharges(chargesOfMessage(res.charges))
        } catch (e) {
            this.holdRefusal(e)
            throw e
        }
    }

    public listenForUpdatesBatch(
        callback: (updates: Update[]) => void,
    ): () => void {
        const id = generateUUID()
        this.updateBatchCallbacks.set(id, callback)
        return () => this.updateBatchCallbacks.delete(id)
    }
}

export function enclosureOf(event: PlanetEvent): Enclosure | undefined {
    if (event.event.case !== "tilesEnclosed") return undefined

    const enclosed = event.event.value
    return {
        countryId: enclosed.countryId,
        closingTile: enclosed.closingTileId,
        wall: [...enclosed.wallTileIds],
        filled: [...enclosed.filledTileIds],
        struck: enclosed.struck.map(({tileId, shields}) => ({tile: tileId, shields})),
        version: versionOf(enclosed.version),
        yours: enclosed.yours || undefined,
    }
}

export function takenOf(enclosure: Enclosure): number[] {
    const struck = new Set(enclosure.struck.map(({tile}) => tile))
    return enclosure.filled.filter((tile) => !struck.has(tile))
}

export function fortificationOf(event: PlanetEvent): Fortification | undefined {
    if (event.event.case !== "landmassFortified") return undefined

    const fortified = event.event.value
    return {landmass: fortified.landmassId, countryId: fortified.countryId, tile: fortified.tileId, version: versionOf(fortified.version)}
}

export function knockOutOf(event: PlanetEvent): KnockOut | undefined {
    if (event.event.case !== "knockOut") return undefined

    const knocked = event.event.value
    return {
        countryId: knocked.countryId,
        byCountryId: knocked.byCountryId,
        tile: knocked.tileId,
        cleared: knocked.clearedTileIds,
        version: versionOf(knocked.version),
    }
}

// The refill and the sizes of the charges may have moved, and only a reading says by how much.
export function rulesChanged(event: PlanetEvent): boolean {
    return event.event.case === "rulesChanged"
}

export function spreadOf(event: PlanetEvent): SpreadClick | undefined {
    if (event.event.case !== "tilesSpread") return undefined

    const spread = event.event.value
    return {countryId: spread.countryId, tile: spread.tileId, spread: [...spread.spreadTileIds]}
}

export function grantedOf(event: PlanetEvent): Charges | undefined {
    if (event.event.case !== "chargesGranted") return undefined

    return chargesOfMessage(event.event.value.charges)
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

export function bombOf(event: PlanetEvent): BombDrop | undefined {
    if (event.event.case !== "bombDropped") return undefined

    const dropped = event.event.value
    return {
        tile: dropped.tileId === 0 ? undefined : dropped.tileId,
        point: {x: dropped.point?.x ?? 0, y: dropped.point?.y ?? 0, z: dropped.point?.z ?? 1},
        countryId: dropped.countryId,
        radius: dropped.radius,
        cleared: dropped.clearedTileIds,
        struck: dropped.struck.map(({tileId, shields}) => ({tile: tileId, shields})),
        version: versionOf(dropped.version),
    }
}

export function asBonusError(e: unknown): unknown {
    if (e instanceof SessionUnavailableError) return e

    if (e instanceof ConnectError) {
        if (e.code === Code.NotFound || e.code === Code.Unimplemented) return new BonusLostError({cause: e})
        if (e.code === Code.FailedPrecondition && e.findDetails(BombRefusal).length > 0) return new BombRefusedError({cause: e})
        if (e.code === Code.Unauthenticated) return new SessionUnavailableError({cause: e})
        if (e.code === Code.PermissionDenied) return bannedOf(e) ?? e
    }

    return e
}

function bannedOf(e: ConnectError): BannedError | undefined {
    const refusal = e.findDetails(BanRefusal)[0]
    return refusal && new BannedError(Number(refusal.bannedUntilUnixMs), {cause: e})
}

export function asShieldError(e: unknown): unknown {
    const locked = allegianceDetailOf(e)
    if (locked) return new AllegianceLockedError(locked, {cause: e})
    if (e instanceof ConnectError && e.code === Code.FailedPrecondition) return new ShieldRefusedError({cause: e})

    return asBonusError(e)
}

export function asRefillError(e: unknown): unknown {
    if (e instanceof ConnectError && e.code === Code.FailedPrecondition) return new BankFullError({cause: e})

    return asBonusError(e)
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

export function allegianceOfMessage(message: AllegianceMessage | undefined): Allegiance | undefined {
    if (!message || message.countryId === "") return undefined

    const allegiance: Allegiance = {countryId: message.countryId}
    const changeableAt = Number(message.changeableAtUnixMs)
    if (changeableAt !== 0) allegiance.changeableAt = changeableAt
    if (message.locked) allegiance.locked = true
    return allegiance
}

export function allegianceDetailOf(e: unknown): Allegiance | undefined {
    if (!(e instanceof ConnectError) || e.code !== Code.FailedPrecondition) return undefined

    return allegianceOfMessage(e.findDetails(AllegianceMessage)[0])
}

export function budgetDetailOf(e: unknown): ClickBudgetMessage | undefined {
    if (!(e instanceof ConnectError)) return undefined

    return e.findDetails(ClickBudgetMessage)[0]
}

export function asClickError(e: unknown): unknown {
    if (e instanceof SessionUnavailableError) return e

    const locked = allegianceDetailOf(e)
    if (locked) return new AllegianceLockedError(locked, {cause: e})

    if (e instanceof ConnectError) {
        if (e.code === Code.FailedPrecondition && e.findDetails(FrozenRefusal).length > 0) return new FrozenError({cause: e})
        if (e.code === Code.FailedPrecondition && e.findDetails(ReachRefusal).length > 0) return new OutOfReachError({cause: e})
        if (e.code === Code.ResourceExhausted) return new RateLimitedError({cause: e})
        if (e.code === Code.PermissionDenied) return bannedOf(e) ?? new VPNBlockedError({cause: e})
        if (e.code === Code.Unauthenticated) return new SessionUnavailableError({cause: e})
    }

    return e
}

const MAP_READS_IN_FLIGHT = 8

export function versionOf(version: MapVersionMessage | undefined): MapVersion {
    return {epoch: version?.epoch ?? 0, seq: Number(version?.seq ?? 0)}
}

export function ownershipsOf(res: GetMapResponse, range: TileRange): Ownerships {
    return {
        range, version: versionOf(res.version), bindings: bindingsOf(res), shields: shieldsOf(res),
        knockedOut: res.knockedOut,
    }
}

export function countsOf(message: GetHoldingsResponse | HoldingsChanged): Counts {
    return {
        version: versionOf(message.version),
        countries: new Map(message.countries.map(({countryId, tiles}) => [countryId, tiles])),
        landmasses: new Map(message.landmasses.map((row: LandmassHoldings) => [
            row.landmassId, new Map(row.holdings.map(({countryId, tiles}) => [countryId, tiles])),
        ])),
    }
}

export function holdingsChangeOf(event: PlanetEvent): Counts | undefined {
    return event.event.case === "holdingsChanged" ? countsOf(event.event.value) : undefined
}

export function heartbeatOf(event: PlanetEvent): MapVersion | undefined {
    return event.event.case === "heartbeat" ? versionOf(event.event.value.version) : undefined
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
        version: versionOf(update.version),
    }
}
