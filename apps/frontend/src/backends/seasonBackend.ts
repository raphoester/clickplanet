import {Code, ConnectError, createPromiseClient, PromiseClient} from "@connectrpc/connect"
import {createConnectTransport} from "@connectrpc/connect-web"
import {SeasonService} from "../gen/grpc/seasons/v1/seasons_connect.ts"
import {Season as SeasonPb} from "../gen/grpc/seasons/v1/seasons_pb.ts"
import {Season, SeasonBackend} from "./season.ts"
import {Config, retrying} from "./transport.ts"

export function newSeasonServiceClient(config: Config): PromiseClient<typeof SeasonService> {
    return createPromiseClient(SeasonService, createConnectTransport({
        baseUrl: config.baseUrl,
        useBinaryFormat: true,
        useHttpGet: true,
        defaultTimeoutMs: config.timeoutMs ?? 5000,
    }))
}

export class ConnectSeasonBackend implements SeasonBackend {
    private read?: Promise<Season | undefined>

    constructor(private readonly client: PromiseClient<typeof SeasonService>, private readonly baseUrl: string) {
    }

    public season(): Promise<Season | undefined> {
        this.read ??= this.current()
        return this.read
    }

    private async current(): Promise<Season | undefined> {
        try {
            const res = await retrying(() => this.client.getSeason({}), "GetSeason")
            return res.season && seasonOf(res.season, this.baseUrl)
        } catch (e) {
            if (e instanceof ConnectError && (e.code === Code.Unimplemented || e.code === Code.NotFound)) return undefined
            throw e
        }
    }
}

function seasonOf(season: SeasonPb, baseUrl: string): Season {
    return {
        number: season.number,
        finaleStartsAt: Number(season.finaleStartsAtUnixMs),
        endsAt: Number(season.endsAtUnixMs),
        finaleFile: `${baseUrl.replace(/\/$/, "")}/seasons/${season.number}/finale.ics`,
    }
}
