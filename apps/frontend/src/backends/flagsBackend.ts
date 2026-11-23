import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {FlagService} from "../gen/grpc/flags/v1/flags_connect.ts"
import type {ListedFlag} from "../domain/flags.ts"
import {FlagLister} from "./flags.ts"
import {Config, retrying} from "./transport.ts"

// A cached GET. The fresh client asks past the browser's copy, which may predate the change it reads for.
export function newFlagServiceClient(config: Config, fresh = false): PromiseClient<typeof FlagService> {
    return createPromiseClient(FlagService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
        fetch: fresh ? (input, init) => globalThis.fetch(input, {...init, cache: "no-cache"}) : undefined,
    }))
}

export class ConnectFlagsBackend implements FlagLister {
    constructor(
        private readonly client: PromiseClient<typeof FlagService>,
        private readonly fresh: PromiseClient<typeof FlagService>,
    ) {
    }

    public async listFlags(fresh: boolean): Promise<ListedFlag[]> {
        const client = fresh ? this.fresh : this.client
        try {
            const res = await retrying(() => client.listFlags({}), "ListFlags")
            return res.flags.map(({code, name, closed}) => ({code, name, closed}))
        } catch (e) {
            // A server from before the flags module opened none.
            if (e instanceof ConnectError && e.code === Code.Unimplemented) return []
            throw e
        }
    }
}
