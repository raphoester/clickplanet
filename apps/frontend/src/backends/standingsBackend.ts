import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {SeasonService} from "../gen/grpc/seasons/v1/seasons_connect.ts"
import {GetMySeasonResponse, Standing as StandingPb} from "../gen/grpc/seasons/v1/seasons_pb.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {MySeason, Standing, StandingsBackend} from "./standings.ts"
import {retrying} from "./transport.ts"

export class ConnectStandingsBackend implements StandingsBackend {
    constructor(
        private readonly client: PromiseClient<typeof SeasonService>,
        private readonly session: SessionProvider,
    ) {
    }

    public async standings(countryCode: string): Promise<Standing[]> {
        try {
            const res = await retrying(() => this.client.getStandings({countryId: countryCode}), "GetStandings")
            return res.standings.map(standingOf)
        } catch (e) {
            if (absent(e)) return []
            throw e
        }
    }

    public async mySeason(): Promise<MySeason | undefined> {
        const token = await this.session.identity()
        if (!token) return undefined

        const headers = new Headers({[SESSION_HEADER]: token})
        try {
            const res = await retrying(() => this.client.getMySeason({}, {headers}), "GetMySeason")
            return mySeasonOf(res)
        } catch (e) {
            if (absent(e) || (e instanceof ConnectError && e.code === Code.Unauthenticated)) return undefined
            throw e
        }
    }
}

function absent(e: unknown): boolean {
    return e instanceof ConnectError && (e.code === Code.Unimplemented || e.code === Code.NotFound)
}

function standingOf(standing: StandingPb): Standing {
    return {
        rank: standing.rank,
        name: standing.name,
        color: standing.color,
        countryCode: standing.countryId,
        tiles: Number(standing.tiles),
    }
}

function mySeasonOf(res: GetMySeasonResponse): MySeason {
    return {
        countryCode: res.countryId || undefined,
        tiles: Number(res.tiles),
        globalRank: res.globalRank || undefined,
        countryRank: res.countryRank || undefined,
    }
}
