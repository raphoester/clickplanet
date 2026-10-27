import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {SeasonService} from "../gen/grpc/scoring/v1/scoring_connect.ts"
import {Season as SeasonPb} from "../gen/grpc/scoring/v1/scoring_pb.ts"
import {Season, SeasonBackend} from "./season.ts"
import {Config, retrying} from "./transport.ts"

export function newSeasonServiceClient(config: Config): PromiseClient<typeof SeasonService> {
    return createPromiseClient(SeasonService, createConnectTransport({
        baseUrl: config.baseUrl,
        interceptors: config.interceptors,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ConnectSeasonBackend implements SeasonBackend {
    private read?: Promise<Season | undefined>

    constructor(private readonly client: PromiseClient<typeof SeasonService>) {
    }

    public season(): Promise<Season | undefined> {
        this.read ??= this.current()
        return this.read
    }

    private async current(): Promise<Season | undefined> {
        try {
            const res = await retrying(() => this.client.getSeason({}), "GetSeason")
            return res.season && seasonOf(res.season)
        } catch (e) {
            if (e instanceof ConnectError && (e.code === Code.Unimplemented || e.code === Code.NotFound)) return undefined
            throw e
        }
    }
}

function seasonOf(season: SeasonPb): Season {
    return {
        number: season.number,
        // A server from before last_day_ends_at sent the same time as finale_starts_at.
        lastDayEndsAt: Number(season.lastDayEndsAtUnixMs || season.finaleStartsAtUnixMs),
        endsAt: Number(season.endsAtUnixMs),
    }
}
