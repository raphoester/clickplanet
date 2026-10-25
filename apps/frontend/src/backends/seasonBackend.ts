import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {SeasonService} from "../gen/grpc/seasons/v1/seasons_connect.ts"
import {Season as SeasonPb} from "../gen/grpc/seasons/v1/seasons_pb.ts"
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

type Seasons = {current?: Season, next?: Season}

export class ConnectSeasonBackend implements SeasonBackend {
    private read?: Promise<Seasons>

    constructor(private readonly client: PromiseClient<typeof SeasonService>) {
    }

    public async season(): Promise<Season | undefined> {
        return (await this.seasons()).current
    }

    public async next(): Promise<Season | undefined> {
        return (await this.seasons()).next
    }

    private seasons(): Promise<Seasons> {
        this.read ??= this.readSeasons()
        return this.read
    }

    private async readSeasons(): Promise<Seasons> {
        try {
            const res = await retrying(() => this.client.getSeason({}), "GetSeason")
            return {current: res.season && seasonOf(res.season), next: res.next && seasonOf(res.next)}
        } catch (e) {
            if (e instanceof ConnectError && (e.code === Code.Unimplemented || e.code === Code.NotFound)) return {}
            throw e
        }
    }
}

function seasonOf(season: SeasonPb): Season {
    const read: Season = {
        number: season.number,
        finaleStartsAt: Number(season.finaleStartsAtUnixMs),
        endsAt: Number(season.endsAtUnixMs),
    }
    return season.startsAtUnixMs === 0n ? read : {...read, startsAt: Number(season.startsAtUnixMs)}
}
