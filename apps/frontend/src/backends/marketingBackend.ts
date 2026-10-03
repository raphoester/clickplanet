import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {SubscriptionService} from "../gen/grpc/marketing/v1/marketing_connect.ts"
import {SubscriptionState as SubscriptionStatePb} from "../gen/grpc/marketing/v1/marketing_pb.ts"
import {MarketingBackend, MarketingError, MarketingFailure, Subscription, SubscriptionState} from "./marketing.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {Config, retrying} from "./transport.ts"

export function newSubscriptionServiceClient(config: Config): PromiseClient<typeof SubscriptionService> {
    return createPromiseClient(SubscriptionService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

const FAILURES: Partial<Record<Code, MarketingFailure>> = {
    [Code.InvalidArgument]: "invalid",
    [Code.PermissionDenied]: "guest",
    [Code.FailedPrecondition]: "alreadySubscribed",
    [Code.Unavailable]: "unavailable",
    [Code.ResourceExhausted]: "tooManyTries",
    [Code.Unauthenticated]: "notSignedIn",
    [Code.Unimplemented]: "off",
    [Code.NotFound]: "off",
}

export class ConnectMarketingBackend implements MarketingBackend {
    constructor(
        private readonly client: PromiseClient<typeof SubscriptionService>,
        private readonly session: SessionProvider,
    ) {
    }

    // No token on purpose: a server that offers season emails refuses the call, one that does not has no route.
    public async offered(): Promise<boolean> {
        try {
            await retrying(() => this.client.getSubscription({}), "GetSubscription")
            return true
        } catch (e) {
            if (e instanceof ConnectError && e.code === Code.Unauthenticated) return true
            if (e instanceof ConnectError && (e.code === Code.Unimplemented || e.code === Code.NotFound)) return false
            throw e
        }
    }

    public async subscription(): Promise<Subscription | undefined> {
        try {
            const res = await this.authenticated((headers) =>
                retrying(() => this.client.getSubscription({}, {headers}), "GetSubscription"))
            return {state: stateOf(res.state), address: res.address}
        } catch (e) {
            if (e instanceof MarketingError && e.failure === "off") return undefined
            throw e
        }
    }

    public async subscribe(address: string): Promise<SubscriptionState> {
        const res = await this.authenticated((headers) => this.client.subscribe({address}, {headers}))
        return stateOf(res.state)
    }

    public async unsubscribe(): Promise<void> {
        await this.authenticated((headers) => this.client.unsubscribe({}, {headers}))
    }

    private async authenticated<T>(call: (headers: Headers) => Promise<T>): Promise<T> {
        try {
            try {
                return await call(await this.headers())
            } catch (e) {
                if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) throw e

                this.session.invalidate()
                return await call(await this.headers())
            }
        } catch (e) {
            const failure = e instanceof ConnectError ? FAILURES[e.code] : undefined
            throw new MarketingError(failure ?? "failed", {cause: e})
        }
    }

    private async headers(): Promise<Headers> {
        const token = await this.session.token()

        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)
        return headers
    }
}

function stateOf(state: SubscriptionStatePb): SubscriptionState {
    switch (state) {
        case SubscriptionStatePb.WAITING:
            return "waiting"
        case SubscriptionStatePb.ACTIVE:
            return "active"
        default:
            return "none"
    }
}
