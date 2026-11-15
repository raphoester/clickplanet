import {createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {BonusService} from "../gen/grpc/bonuses/v1/bonuses_connect.ts"
import {Config} from "./transport.ts"

// What the bonuses module grants: the page reads its sign-in refill rule. The boxes are BoxesBackend's.
export function newBonusServiceClient(config: Config): PromiseClient<typeof BonusService> {
    return createPromiseClient(BonusService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}
