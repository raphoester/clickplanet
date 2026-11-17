import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {v4 as generateUUID} from "uuid"
import {BoxService} from "../gen/grpc/boxes/v1/boxes_connect.ts"
import {BoxEvent} from "../gen/grpc/boxes/v1/boxes_pb.ts"
import {BonusReward} from "../domain/bonus.ts"
import {BoxCatcher, BoxOffer, GrantSource, SessionSource} from "./backend.ts"
import {now as budgetNow} from "./clickBudget.ts"
import {asBonusError} from "./planetBackend.ts"
import {NoSession, SESSION_HEADER, SessionProvider} from "./session.ts"
import {Config, NO_TIMEOUT, openStream} from "./transport.ts"

// A box's reward is a grant planet tells a moment after the catch; past this, none comes.
export const BOX_REWARD_WAIT_MS = 2500

export function newBoxServiceClient(config: Config): PromiseClient<typeof BoxService> {
    return createPromiseClient(BoxService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export function boxOf(event: BoxEvent, at = budgetNow(), wallClock = Date.now()): BoxOffer | undefined {
    if (event.event.case !== "boxOffered") return undefined

    const offered = event.event.value

    return {
        token: offered.token,
        seed: offered.seed,
        expiresAt: at + (Number(offered.expiresAtUnixMs) - wallClock),
    }
}

// The boxes offered to the page, on a stream of their own. The boxes pay nothing: what a box held is a grant on
// planet's stream, which the grants say.
export class BoxesBackend implements BoxCatcher {
    private readonly boxCallbacks = new Map<string, (offer: BoxOffer) => void>()
    private stopBoxes: () => void
    private readonly stopFollowing: () => void
    private streamToken: string | undefined

    constructor(
        private client: PromiseClient<typeof BoxService>,
        private grants: GrantSource,
        sessions: SessionSource,
        private session: SessionProvider = new NoSession(),
    ) {
        this.stopBoxes = this.openBoxStream()
        this.stopFollowing = sessions.listenForSessions((token) => this.followSession(token))
        void this.followIdentity()
    }

    public close() {
        this.stopFollowing()
        this.stopBoxes()
        this.boxCallbacks.clear()
    }

    // Opened as the planet's is, and beside it: a box goes only to a caller with it open.
    private openBoxStream(): () => void {
        return openStream(
            (signal) => {
                const token = this.session.heldIdentity()
                this.streamToken = token

                const headers = new Headers()
                if (token) headers.set(SESSION_HEADER, token)

                return this.client.listenForEvents({}, {signal, headers, timeoutMs: NO_TIMEOUT})
            },
            (event) => {
                const box = boxOf(event)
                if (box) this.boxCallbacks.forEach(callback => callback(box))
            },
            "box events",
        )
    }

    private async followIdentity(): Promise<void> {
        this.followSession(await this.session.identity())
    }

    // The stream reads the token only when it opens: a new one, as a guest signs in, opens it again as that account.
    private followSession(token: string | undefined): void {
        if (!token || token === this.streamToken) return

        this.streamToken = token
        this.stopBoxes()
        this.stopBoxes = this.openBoxStream()
    }

    public listenForBoxes(onOffered: (offer: BoxOffer) => void): () => void {
        const id = generateUUID()
        this.boxCallbacks.set(id, onOffered)

        return () => this.boxCallbacks.delete(id)
    }

    public async claimBox(token: string): Promise<BonusReward | undefined> {
        const reward = awaitReward(this.grants)
        try {
            await this.claimRetried(token)
        } catch (e) {
            reward.cancel()
            throw e
        }
        return reward.within(BOX_REWARD_WAIT_MS)
    }

    private async claimRetried(token: string): Promise<void> {
        try {
            await this.claim(token)
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) throw asBonusError(e)

            this.session.invalidate()

            try {
                await this.claim(token)
            } catch (retried) {
                throw asBonusError(retried)
            }
        }
    }

    private async claim(token: string): Promise<void> {
        const sessionToken = await this.session.token()

        const headers = new Headers()
        if (sessionToken) headers.set(SESSION_HEADER, sessionToken)

        // No `retrying`: the first request that lands spends the token.
        await this.client.claimBox({token}, {headers})
        this.followSession(sessionToken)
    }
}

// The first grant planet tells once the catch is sent is the box's: the bonuses module pays it about a second later.
function awaitReward(grants: GrantSource): {within: (ms: number) => Promise<BonusReward | undefined>, cancel: () => void} {
    let settle: (reward: BonusReward | undefined) => void = () => {}
    const rewarded = new Promise<BonusReward | undefined>((resolve) => settle = resolve)
    const stop = grants.listenForGrants(([reward]) => {
        if (reward) settle(reward)
    })
    const cancel = () => {
        stop()
        settle(undefined)
    }

    return {
        within: (ms) => {
            const timer = setTimeout(cancel, ms)
            return rewarded.finally(() => {
                clearTimeout(timer)
                stop()
            })
        },
        cancel,
    }
}
