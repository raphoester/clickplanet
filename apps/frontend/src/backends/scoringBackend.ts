import {createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {ScoringService} from "../gen/grpc/scoring/v1/scoring_connect.ts"
import {Config} from "./transport.ts"

export function newScoringServiceClient(config: Config): PromiseClient<typeof ScoringService> {
    return createPromiseClient(ScoringService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}
