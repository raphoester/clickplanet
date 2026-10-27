import {ClosedRound} from "../gen/grpc/scoring/v1/scoring_pb.ts"
import {Day} from "../domain/clip/story.ts"

const PODIUM = 3

// The day a race closed last, as a clip tells it: the places of its podium that scored, ties sharing one, and the
// season's table after it.
export function dayOf(closed: ClosedRound | undefined): Day | undefined {
    if (closed === undefined) return undefined
    return {
        season: closed.season,
        number: closed.number,
        finale: closed.finale,
        own: closed.own,
        endedAt: Number(closed.endedAtUnixMs),
        podium: closed.standings
            .filter(({rank, points}) => rank <= PODIUM && points > 0)
            .sort((a, b) => a.rank - b.rank)
            .map(({rank, countryId, points}) => ({country: countryId, rank, points})),
        table: closed.after.map(({rank, countryId, points}) => ({country: countryId, rank, points})),
    }
}
