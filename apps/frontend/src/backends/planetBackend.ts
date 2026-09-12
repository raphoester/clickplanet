import {
    BombDrop,
    Bomber,
    BonusCatch,
    BonusHandlers,
    BonusListener,
    GlobePoint,
    BonusLostError,
    BonusOffer,
    Enclosure,
    Ownerships,
    OwnershipsGetter,
    RateLimitedError,
    TileClicker,
    Update,
    UpdatesListener,
    VPNBlockedError,
} from "./backend.ts";
import {BonusReward} from "../domain/bonus.ts";
import {
    BonusKind,
    ClickBudget as ClickBudgetMessage,
    GetMapResponse,
    PlanetEvent,
} from "../gen/grpc/planet/v1/planet_pb.ts";
import {ClickBudget, ClickBudgetSource, now as budgetNow} from "./clickBudget.ts";
import {ClickService} from "../gen/grpc/planet/v1/planet_connect.ts";
import {SessionService} from "../gen/grpc/session/v1/session_connect.ts";
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

/**
 * Minting is a POST that must not be cached and is not on the click path's
 * critical timing, so it takes neither of the click transport's two options.
 */
export function newSessionServiceClient(config: Config): PromiseClient<typeof SessionService> {
    return createPromiseClient(SessionService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class PlanetBackend implements TileClicker, OwnershipsGetter, UpdatesListener, ClickBudgetSource, BonusListener, Bomber {
    private pendingUpdates: Update[] = []
    private readonly updateBatchCallbacks = new Map<string, (updates: Update[]) => void>()
    private readonly updateCallbacks = new Map<string, (update: Update) => void>()
    private readonly bonusCallbacks = new Map<string, BonusHandlers>()
    private readonly bombCallbacks = new Map<string, (drop: BombDrop) => void>()
    private readonly budgetCallbacks = new Map<string, (budget: ClickBudget) => void>()
    private readonly flushTimer: ReturnType<typeof setInterval>
    private readonly stopListening: () => void

    /** The last reading the server sent, before this client's own clicks. */
    private budgetAnchor: ClickBudget | undefined

    /** Clicks sent and not yet answered — see reportBudget. */
    private inFlight = 0

    /** Re-reads the allowance when a caught bonus runs out — see claim. */
    private bonusEndTimer: ReturnType<typeof setTimeout> | undefined

    constructor(
        private client: PromiseClient<typeof ClickService>,
        batchUpdateDurationMs: number,
        private session: SessionProvider = new NoSession(),
    ) {
        void this.readBudget()

        // One stream, every case. The tile feed and the bonus feed ride the same
        // connection because they are cases of one `oneof` — opening a second
        // stream for the second feed is exactly what the envelope exists to
        // avoid, and would cost every client a second connection.
        this.stopListening = this.openEventStream()

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
        clearTimeout(this.bonusEndTimer)
        this.stopListening()
        this.updateBatchCallbacks.clear()
        this.updateCallbacks.clear()
        this.bonusCallbacks.clear()
        this.bombCallbacks.clear()
        this.budgetCallbacks.clear()
        this.pendingUpdates = []
    }

    /**
     * A click the server refused for its session is retried once against a
     * freshly minted one, and the player never learns it happened: a token that
     * lapsed mid-session, or one bound to an address that changed when a phone
     * moved onto cellular, is not something to raise a dialog about. The retry
     * is not a loop — a second refusal is reported.
     */
    public async clickTile(tileId: number, countryId: string) {
        this.inFlight++
        this.reportBudget()

        try {
            await this.click(tileId, countryId)
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) {
                throw asClickError(e)
            }

            this.session.invalidate()

            try {
                await this.click(tileId, countryId)
            } catch (retried) {
                throw asClickError(retried)
            }
        } finally {
            this.inFlight--
            this.reportBudget()
        }
    }

    private async click(tileId: number, countryId: string): Promise<void> {
        const token = await this.session.token()

        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)

        try {
            const res = await retrying(
                () => this.client.click({tileId, countryId}, {headers}),
                `click ${tileId}`,
            )
            this.anchorBudget(res.budget)
        } catch (e) {
            // A refusal carries the reading on the error, because there is no
            // answer to put it in — and it is the refusal the counter most has
            // to agree with.
            this.anchorBudget(budgetDetailOf(e))
            throw e
        }
    }

    /**
     * Asked at load, and when a caught bonus ends. Everything else is learned
     * from the answers to this client's own clicks.
     *
     * A server too old to answer leaves the counter off rather than breaking
     * the page: the frontend deploys separately from the backend.
     */
    private async readBudget(): Promise<void> {
        try {
            const res = await this.client.getBudget({})
            this.anchorBudget(res.budget)
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unimplemented) return
            console.error("could not read the click budget", e)
        }
    }

    private anchorBudget(budget: ClickBudgetMessage | undefined): void {
        // A server with no throttle says nothing, and the counter stays hidden
        // rather than claiming an allowance nobody is enforcing.
        if (!budget || budget.capacity === 0) return

        this.budgetAnchor = {
            tokens: budget.tokens,
            capacity: budget.capacity,
            perSecond: budget.refillPerSecond,
            readAt: budgetNow(),
        }

        this.reportBudget()
    }

    /**
     * Publishes the anchor with this client's own clicks taken off it.
     *
     * `readAt` deliberately stays the server's reading rather than becoming
     * now: the refill since then is real and still owed, and subtracting a
     * click in flight from a reading is the same arithmetic the server will do
     * when that click lands. The result only ever *under*-promises, which is
     * the side to be wrong on — a counter that says 1 and is refused is a bug
     * the player sees, and one that says 0 and works is a click they still get.
     */
    private reportBudget(): void {
        const anchor = this.budgetAnchor
        if (!anchor) return

        const budget = {...anchor, tokens: anchor.tokens - this.inFlight}
        this.budgetCallbacks.forEach(callback => callback(budget))
    }

    public watchClickBudget(callback: (budget: ClickBudget) => void): () => void {
        const id = generateUUID()
        this.budgetCallbacks.set(id, callback)

        // The load-time read usually lands before anything subscribes.
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

            callback({bindings: bindingsOf(res)})
        }
    }

    /**
     * Opens the one stream and hands each frame to whichever feed it belongs to.
     *
     * A case this build does not know falls through all of them, which is what
     * lets the backend add an event type without breaking a deployed client.
     */
    private openEventStream(): () => void {
        return openStream(
            (signal) => this.client.listenForEvents({}, {signal, timeoutMs: NO_TIMEOUT}),
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

                const taken = catchOf(event)
                if (taken) {
                    this.bonusCallbacks.forEach(handlers => handlers.onTaken(taken))
                    return
                }

                const drop = bombOf(event)
                if (drop) {
                    // Tile updates wait for the next batch; the ones that came
                    // before this blast on the wire have to land before it too,
                    // or a tile taken just before the bomb repaints over the crater.
                    this.flushUpdates()
                    this.bombCallbacks.forEach(callback => callback(drop))
                    return
                }

                const enclosure = enclosureOf(event)
                if (enclosure) this.bonusCallbacks.forEach(handlers => handlers.onEnclosed(enclosure))
            },
            "planet events",
        )
    }

    public listenForUpdates(callback: (update: Update) => void): () => void {
        const id = generateUUID()
        this.updateCallbacks.set(id, callback)

        return () => this.updateCallbacks.delete(id)
    }

    public listenForBonuses(handlers: BonusHandlers): () => void {
        const id = generateUUID()
        this.bonusCallbacks.set(id, handlers)

        return () => this.bonusCallbacks.delete(id)
    }

    /**
     * Redeems a box, with the same one-shot session retry a click gets: a token
     * that lapsed mid-session is not worth losing the box over.
     */
    public async claimBonus(token: string, countryId: string): Promise<BonusReward> {
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

    private async claim(token: string, countryId: string): Promise<BonusReward> {
        const sessionToken = await this.session.token()

        const headers = new Headers()
        if (sessionToken) headers.set(SESSION_HEADER, sessionToken)

        // Deliberately not wrapped in `retrying`: a claim is not idempotent —
        // the token is spent on the first one that lands, so a retry of a
        // request whose answer was lost reports the box as lost when it was in
        // fact won.
        const res = await this.client.claimBonus({token, countryId}, {headers})

        // The allowance arrives widened on the answer, so the meter follows the
        // server's own policy rather than a multiplication done here.
        this.anchorBudget(res.budget)

        // Only a kind this build knows is ever drawn, so only one can be caught.
        const reward = rewardOf(res.kind, res.durationSeconds, {blastRadius: res.blastRadius, shapes: res.enclosures, maxTiles: res.enclosureMaxTiles})
        if (!reward) throw new BonusLostError()

        // The widened reading says nothing about when the widening stops, so
        // left alone the meter keeps replaying a burst of 30 until the next
        // click re-anchors it. Ask again once it is over. The server started
        // the bonus before it answered, so this always lands after its end.
        clearTimeout(this.bonusEndTimer)
        this.bonusEndTimer = setTimeout(() => void this.readBudget(), reward.seconds * 1000)

        return reward
    }

    public listenForBombs(onDropped: (drop: BombDrop) => void): () => void {
        const id = generateUUID()
        this.bombCallbacks.set(id, onDropped)

        return () => this.bombCallbacks.delete(id)
    }

    /** With the same one-shot session retry a claim gets: a bomb is worth keeping. */
    public async dropBomb(target: GlobePoint, countryId: string): Promise<void> {
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

        // Not wrapped in `retrying`, for the reason a claim is not: the bomb is
        // spent by the first request that lands.
        await this.client.dropBomb({target, countryId}, {headers})
    }

    public listenForUpdatesBatch(
        callback: (updates: Update[]) => void,
    ): () => void {
        const id = generateUUID()
        this.updateBatchCallbacks.set(id, callback)
        return () => this.updateBatchCallbacks.delete(id)
    }
}

/**
 * Reads the box this client was offered.
 *
 * The deadline is rebuilt from how long the server said was **left** rather
 * than from the timestamp it sent: the two wall clocks are unrelated, and a
 * client whose clock is a minute fast would otherwise treat every box as
 * already lapsed.
 */
export function offerOf(event: PlanetEvent, at = budgetNow(), wallClock = Date.now()): BonusOffer | undefined {
    if (event.event.case !== "bonusOffered") return undefined

    const offered = event.event.value

    // A kind this build does not know is a box it cannot describe, so it is not
    // drawn at all rather than drawn as something it is not.
    const reward = rewardOf(offered.kind, offered.durationSeconds)
    if (!reward) return undefined

    return {
        token: offered.token,
        seed: offered.seed,
        reward,
        expiresAt: at + (Number(offered.expiresAtUnixMs) - wallClock),
    }
}

export function catchOf(event: PlanetEvent): BonusCatch | undefined {
    if (event.event.case !== "bonusTaken") return undefined

    return {countryId: event.event.value.countryId}
}

/**
 * Somebody closed a shape. `yours` is only read when the server marked it so:
 * how many shapes somebody else has left is not sent, and a zero here would
 * read as "your bonus is over".
 */
export function enclosureOf(event: PlanetEvent): Enclosure | undefined {
    if (event.event.case !== "tilesEnclosed") return undefined

    const enclosed = event.event.value
    return {
        countryId: enclosed.countryId,
        closingTile: enclosed.closingTileId,
        wall: [...enclosed.wallTileIds],
        filled: [...enclosed.filledTileIds],
        yours: enclosed.yours ? {shapesLeft: enclosed.enclosuresLeft} : undefined,
    }
}

/**
 * An offer carries neither a blast radius nor an enclose bonus's shapes; only the
 * answer to a claim does, which is when they are needed.
 */
function rewardOf(
    kind: BonusKind,
    seconds: number,
    {blastRadius = 0, shapes = 0, maxTiles = 0}: {blastRadius?: number, shapes?: number, maxTiles?: number} = {},
): BonusReward | undefined {
    switch (kind) {
        case BonusKind.TRIPLE_CLICKS:
            return {kind: "tripleClicks", seconds}
        case BonusKind.SPREAD_CLICKS:
            return {kind: "spreadClicks", seconds}
        case BonusKind.BOMB:
            return {kind: "bomb", seconds, radius: blastRadius}
        case BonusKind.ENCLOSE_CLICKS:
            return {kind: "encloseClicks", seconds, shapes, maxTiles}
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
    }
}

export function asBonusError(e: unknown): unknown {
    if (e instanceof SessionUnavailableError) return e

    if (e instanceof ConnectError) {
        // NotFound is the box being gone; Unimplemented is a server with boxes
        // switched off, which a client that drew one can still meet after a
        // deploy. Both mean the same thing to the player: it got away.
        if (e.code === Code.NotFound || e.code === Code.Unimplemented) return new BonusLostError({cause: e})
        if (e.code === Code.Unauthenticated) return new SessionUnavailableError({cause: e})
    }

    return e
}

export function budgetDetailOf(e: unknown): ClickBudgetMessage | undefined {
    if (!(e instanceof ConnectError)) return undefined

    return e.findDetails(ClickBudgetMessage)[0]
}

export function asClickError(e: unknown): unknown {
    if (e instanceof SessionUnavailableError) return e

    if (e instanceof ConnectError) {
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

/**
 * Anything that is not a tile update is dropped, heartbeats included. An event
 * case this build does not know reads as an unset `oneof` and lands here too,
 * which is what lets the backend add one without breaking a deployed client.
 */
export function updateOf(event: PlanetEvent): Update | undefined {
    if (event.event.case !== "tileUpdate") return undefined

    const update = event.event.value
    return {
        tile: update.tileId,
        previousCountry: update.previousCountryId === "" ? undefined : update.previousCountryId,
        newCountry: update.countryId,
    }
}
