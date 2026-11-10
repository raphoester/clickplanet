import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {AntibotService} from "../gen/grpc/antibot/v1/antibot_connect.ts"
import {WarningAcknowledger, WarningsListener} from "./antibot.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {Config, NO_TIMEOUT, openStream, retryingAtOnce} from "./transport.ts"

export function newAntibotServiceClient(config: Config): PromiseClient<typeof AntibotService> {
    return createPromiseClient(AntibotService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ConnectAntibotBackend implements WarningsListener, WarningAcknowledger {
    constructor(
        private readonly client: PromiseClient<typeof AntibotService>,
        private readonly session: SessionProvider,
    ) {
    }

    public heldIdentity(): string | undefined {
        return this.session.heldIdentity()
    }

    public listenForWarnings(onWarned: () => void): () => void {
        const client = this.client
        const session = this.session
        return openStream(
            (signal) => {
                const headers = new Headers()
                const token = session.heldIdentity()
                if (token) headers.set(SESSION_HEADER, token)
                return client.listenForEvents({}, {signal, timeoutMs: NO_TIMEOUT, headers})
            },
            (event) => {
                if (event.event.case === "warning") onWarned()
            },
            "warnings",
            {stopsOn: (e) => e instanceof ConnectError && e.code === Code.Unauthenticated},
        )
    }

    public async acknowledgeWarning(): Promise<void> {
        const headers = new Headers()
        const token = await this.session.identity()
        if (token) headers.set(SESSION_HEADER, token)
        await retryingAtOnce(() => this.client.acknowledgeWarning({}, {headers}), "AcknowledgeWarning")
    }
}
