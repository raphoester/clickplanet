import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {SeasonService} from "../gen/grpc/seasons/v1/seasons_connect.ts"
import {
    GetMySeasonResponse,
    Race as RacePb,
    RoundStanding as RoundStandingPb,
    Score as ScorePb,
    Standing as StandingPb,
} from "../gen/grpc/seasons/v1/seasons_pb.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {MySeason, Race, RoundStanding, Score, Standing, StandingsBackend} from "./standings.ts"
import {titleOf} from "./title.ts"
import {NO_TIMEOUT, openStream, retrying} from "./transport.ts"

export class ConnectStandingsBackend implements StandingsBackend {
    constructor(
        private readonly client: PromiseClient<typeof SeasonService>,
        private readonly session: SessionProvider,
    ) {
    }

    public listenForStandings(countryCode: string, onStandings: (standings: Standing[]) => void): () => void {
        const client = this.client
        return openStream(
            (signal) => client.listenForEvents({countryId: countryCode}, {signal, timeoutMs: NO_TIMEOUT}),
            (event) => {
                if (event.event.case === "board") onStandings(event.event.value.standings.map(standingOf))
            },
            "standings",
        )
    }

    public listenForRace(onRace: (race: Race) => void): () => void {
        const client = this.client
        return openStream(
            (signal) => client.listenForEvents({countryId: ""}, {signal, timeoutMs: NO_TIMEOUT}),
            (event) => {
                if (event.event.case === "race") onRace(raceOf(event.event.value))
            },
            "race",
        )
    }

    public async mySeason(countryCode: string): Promise<MySeason | undefined> {
        const token = await this.session.identity()
        if (!token) return undefined

        const headers = new Headers({[SESSION_HEADER]: token})
        try {
            const res = await retrying(() => this.client.getMySeason({countryId: countryCode}, {headers}), "GetMySeason")
            return mySeasonOf(res, countryCode)
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
        wornTitle: titleOf(standing.wornTitle),
    }
}

function raceOf(race: RacePb): Race {
    return {
        round: race.round && {
            number: race.round.number,
            endsAt: Number(race.round.endsAtUnixMs),
            finale: race.round.finale,
            standings: race.round.standings.map(roundStandingOf),
        },
        scores: race.scores.map(scoreOf),
        closed: race.closed && {
            season: race.closed.season,
            number: race.closed.number,
            endedAt: Number(race.closed.endedAtUnixMs),
            finale: race.closed.finale,
            standings: race.closed.standings.map(roundStandingOf),
            before: race.closed.before.map(scoreOf),
            after: race.closed.after.map(scoreOf),
        },
    }
}

function roundStandingOf(standing: RoundStandingPb): RoundStanding {
    return {rank: standing.rank, countryCode: standing.countryId, share: standing.share, points: standing.points}
}

function scoreOf(score: ScorePb): Score {
    return {rank: score.rank, countryCode: score.countryId, points: score.points, roundsWon: score.roundsWon}
}

function mySeasonOf(res: GetMySeasonResponse, countryCode: string): MySeason {
    const wornTitle = titleOf(res.wornTitle)
    if (countryCode !== "") return {countryCode, tiles: Number(res.countryTiles), rank: res.countryRank || undefined, wornTitle}
    return {countryCode: res.countryId || undefined, tiles: Number(res.tiles), rank: res.globalRank || undefined, wornTitle}
}
