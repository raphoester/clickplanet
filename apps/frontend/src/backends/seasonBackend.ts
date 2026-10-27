import {createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {SeasonService} from "../gen/grpc/scoring/v1/scoring_connect.ts"
import {Config} from "./transport.ts"

export function newSeasonServiceClient(config: Config): PromiseClient<typeof SeasonService> {
    return createPromiseClient(SeasonService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}
