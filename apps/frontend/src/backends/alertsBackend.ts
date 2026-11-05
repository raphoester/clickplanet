import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {AlertsService} from "../gen/grpc/alerts/v1/alerts_connect.ts"
import {AlertsBackend, DiscordLink, NOT_OFFERED} from "./alerts.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {Config, retrying} from "./transport.ts"

export function newAlertsServiceClient(config: Config): PromiseClient<typeof AlertsService> {
    return createPromiseClient(AlertsService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ConnectAlertsBackend implements AlertsBackend {
    constructor(
        private readonly client: PromiseClient<typeof AlertsService>,
        private readonly session: SessionProvider,
    ) {
    }

    public async discordLink(): Promise<DiscordLink> {
        try {
            const res = await this.authenticated((headers) =>
                retrying(() => this.client.getDiscordLink({}, {headers}), "GetDiscordLink"))
            return {offered: res.offered, linked: res.linked, serverUrl: res.serverUrl}
        } catch (e) {
            if (e instanceof ConnectError && (e.code === Code.Unimplemented || e.code === Code.NotFound)) return NOT_OFFERED
            throw e
        }
    }

    private async authenticated<T>(call: (headers: Headers) => Promise<T>): Promise<T> {
        try {
            return await call(await this.headers())
        } catch (e) {
            if (!(e instanceof ConnectError) || e.code !== Code.Unauthenticated) throw e

            this.session.invalidate()
            return await call(await this.headers())
        }
    }

    private async headers(): Promise<Headers> {
        const token = await this.session.token()

        const headers = new Headers()
        if (token) headers.set(SESSION_HEADER, token)
        return headers
    }
}
