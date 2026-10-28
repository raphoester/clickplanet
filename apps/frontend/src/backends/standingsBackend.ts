import {Code, ConnectError, PromiseClient} from "@connectrpc/connect"
import {ScoringService} from "../gen/grpc/scoring/v1/scoring_connect.ts"
import {
    GetMyTallyResponse,
    Round as RoundPb,
    RoundStanding as RoundStandingPb,
    Score as ScorePb,
    Standing as StandingPb,
} from "../gen/grpc/scoring/v1/scoring_pb.ts"
import {SESSION_HEADER, SessionProvider} from "./session.ts"
import {MySeason, RaceListener, Round, RoundStanding, Score, Standing, StandingsBackend} from "./standings.ts"
import {titleOf} from "./title.ts"
import {NO_TIMEOUT, openStream, retrying} from "./transport.ts"

export class ConnectStandingsBackend implements StandingsBackend {
    constructor(
        private readonly client: PromiseClient<typeof ScoringService>,
        private readonly session: SessionProvider,
    ) {
    }

    public listenForStandings(bookId: string, countryCode: string, onStandings: (standings: Standing[]) => void): () => void {
        const client = this.client
        return openStream(
            (signal) => client.listenForEvents({bookId, countryId: countryCode}, {signal, timeoutMs: NO_TIMEOUT}),
            (event) => {
                if (event.event.case === "board") onStandings(event.event.value.standings.map(standingOf))
            },
            "standings",
        )
    }

    public listenForRace(bookId: string, roundIds: readonly string[], {onScores, onRound}: RaceListener): () => void {
        const client = this.client
        return openStream(
            (signal) => client.listenForEvents({bookId, countryId: "", roundIds: [...roundIds]}, {signal, timeoutMs: NO_TIMEOUT}),
            (event) => {
                switch (event.event.case) {
                    case "scores":
                        onScores(event.event.value.scores.map(scoreOf))
                        return
                    case "round":
                        onRound(roundOf(event.event.value))
                        return
                    case "board":
                    case "heartbeat":
                    case undefined:
                        return
                }
            },
            "race",
        )
    }

    public async mySeason(bookId: string, countryCode: string): Promise<MySeason | undefined> {
        const token = await this.session.identity()
        if (!token) return undefined

        const headers = new Headers({[SESSION_HEADER]: token})
        try {
            const res = await retrying(() => this.client.getMyTally({bookId, countryId: countryCode}, {headers}), "GetMyTally")
            return mySeasonOf(res, countryCode)
        } catch (e) {
            if (absent(e) || (e instanceof ConnectError && e.code === Code.Unauthenticated)) return undefined
            throw e
        }
    }

    public async round(roundId: string): Promise<Round | undefined> {
        try {
            const res = await retrying(() => this.client.getRound({roundId}), "GetRound")
            return res.round && roundOf(res.round)
        } catch (e) {
            if (absent(e)) return undefined
            throw e
        }
    }

    public async scores(bookId: string): Promise<Score[]> {
        try {
            const res = await retrying(() => this.client.getScores({bookId}), "GetScores")
            return res.scores?.scores.map(scoreOf) ?? []
        } catch (e) {
            if (absent(e)) return []
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
        accountId: standing.accountId,
        name: standing.name,
        color: standing.color,
        countryCode: standing.countryId,
        tiles: Number(standing.tiles),
        wornTitle: titleOf(standing.wornTitle),
    }
}

function roundOf(round: RoundPb): Round {
    return {
        roundId: round.roundId,
        times: round.times,
        startedAt: Number(round.startedAtUnixMs),
        endedAt: Number(round.endedAtUnixMs),
        standings: round.standings.map(roundStandingOf),
        takes: round.takes.map((takes) => ({countryCode: takes.countryId, tiles: Number(takes.tiles)})),
        players: round.players.map(standingOf),
        before: round.before.map(scoreOf),
        after: round.after.map(scoreOf),
    }
}

function roundStandingOf(standing: RoundStandingPb): RoundStanding {
    return {rank: standing.rank, countryCode: standing.countryId, share: standing.share, points: standing.points}
}

function scoreOf(score: ScorePb): Score {
    return {rank: score.rank, countryCode: score.countryId, points: score.points, roundsWon: score.roundsWon}
}

function mySeasonOf(res: GetMyTallyResponse, countryCode: string): MySeason {
    const wornTitle = titleOf(res.wornTitle)
    if (countryCode !== "") return {countryCode, tiles: Number(res.countryTiles), rank: res.countryRank || undefined, wornTitle}
    return {countryCode: res.countryId || undefined, tiles: Number(res.tiles), rank: res.globalRank || undefined, wornTitle}
}
