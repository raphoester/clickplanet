import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {PromoService} from "../gen/grpc/promo/v1/promo_connect.ts"
import {PromoBackend, TeaserCard} from "./promo.ts"
import {Config, retrying} from "./transport.ts"

export function newPromoServiceClient(config: Config): PromiseClient<typeof PromoService> {
    return createPromiseClient(PromoService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        // A cached GET: the server keeps a teaser a minute in Cloudflare.
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ConnectPromoBackend implements PromoBackend {
    constructor(private readonly client: PromiseClient<typeof PromoService>) {
    }

    public async teaser(battleId: string): Promise<TeaserCard[]> {
        try {
            const res = await retrying(() => this.client.getTeaser({battleId}), "GetTeaser")
            return res.cards.map(({backgroundUrl, headline, line}) => ({backgroundUrl, headline, line}))
        } catch (e) {
            if (e instanceof ConnectError && [Code.Unimplemented, Code.NotFound].includes(e.code)) return []
            throw e
        }
    }
}
