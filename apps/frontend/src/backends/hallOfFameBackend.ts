import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {SeasonService} from "../gen/grpc/scoring/v1/scoring_connect.ts"
import {EndedSeason as EndedSeasonPb} from "../gen/grpc/scoring/v1/scoring_pb.ts"
import {EndedSeason, HallOfFameBackend} from "./hallOfFame.ts"
import {scoreOf, standingOf} from "./standingsBackend.ts"
import {retrying} from "./transport.ts"

export class ConnectHallOfFameBackend implements HallOfFameBackend {
    constructor(private readonly client: PromiseClient<typeof SeasonService>) {
    }

    public async hallOfFame(): Promise<EndedSeason[]> {
        try {
            const res = await retrying(() => this.client.getHallOfFame({}), "GetHallOfFame")
            return res.seasons.map(endedSeasonOf)
        } catch (e) {
            if (e instanceof ConnectError && (e.code === Code.Unimplemented || e.code === Code.NotFound)) return []
            throw e
        }
    }
}

function endedSeasonOf(season: EndedSeasonPb): EndedSeason {
    return {
        number: season.number,
        endedAt: Number(season.endedAtUnixMs),
        scores: season.scores.map(scoreOf),
        standings: season.standings.map(standingOf),
    }
}
