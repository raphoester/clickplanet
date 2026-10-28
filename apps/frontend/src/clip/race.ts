import {Debrief, Timeline} from "../gen/grpc/gamemaster/v1/gamemaster_pb.ts"
import {Round} from "../gen/grpc/scoring/v1/scoring_pb.ts"
import {Day} from "../domain/clip/story.ts"

const PODIUM = 3

type Named = {day: Pick<Day, "season" | "number" | "finale" | "own">, roundId: string}

// The moment scoring scored last in the game's words, and the round to read its numbers by. A season's end is its
// finale's round, numbered as its battle, so it keys as the finale's BattleEnded does.
function namedOf(debrief: Debrief | undefined): Named | undefined {
    const moment = debrief?.moment
    switch (moment?.case) {
        case undefined:
            return undefined
        case "day":
            return {day: {season: moment.value.season, number: moment.value.day, finale: false, own: false}, roundId: moment.value.roundId}
        case "battle":
            return {
                day: {season: moment.value.season, number: moment.value.number, finale: moment.value.endsSeason, own: true},
                roundId: moment.value.roundId,
            }
        case "season": {
            const battle = moment.value.finaleBattle
            return {
                day: {season: moment.value.season, number: battle?.number ?? 0, finale: true, own: true},
                roundId: moment.value.finaleRoundId,
            }
        }
    }
}

// The round the gamemaster debriefed last, as a clip tells it: the places of its podium that scored, ties sharing one,
// and the season's table after it. None when the round read is another one.
export function dayOf(debrief: Debrief | undefined, round: Round | undefined): Day | undefined {
    const named = namedOf(debrief)
    if (named === undefined || round === undefined || named.roundId === "" || round.roundId !== named.roundId) return undefined
    return {
        ...named.day,
        endedAt: Number(round.endedAtUnixMs),
        podium: round.standings
            .filter(({rank, points}) => rank <= PODIUM && points > 0)
            .sort((a, b) => a.rank - b.rank)
            .map(({rank, countryId, points}) => ({country: countryId, rank, points})),
        table: round.after.map(({rank, countryId, points}) => ({country: countryId, rank, points})),
    }
}

// When the season's finale starts: the battle to come that runs to the season's end.
export function finaleAtOf(timeline: Timeline | undefined): number | undefined {
    return timeline?.battles.find(({endsSeason}) => endsSeason)?.startsAt?.toDate().getTime()
}
