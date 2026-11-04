import {afterEach, describe, expect, it} from "vitest"
import {Round, Score} from "../../backends/standings.ts"
import {useRace} from "./raceStore.ts"

const SCORES: Score[] = [{rank: 1, countryCode: "fr", points: 43, roundsWon: 1}]

const roundOf = (roundId: string, endedAt = 0): Round =>
    ({roundId, times: 1, startedAt: 0, endedAt, standings: [], takes: [], players: [], before: [], after: []})

const race = () => useRace.getState().race

afterEach(() => useRace.setState({race: undefined, heard: undefined}))

describe("the race", () => {
    it("is told once the book's points come, with today's round and the battle's while they are open", () => {
        const {follow, hearScores, hearRound} = useRace.getState()
        follow("season-0", "season-0/day-5", "entry-1")
        hearRound(roundOf("season-0/day-5"))
        expect(race()).toBeUndefined()

        hearScores(SCORES)
        expect(race()).toEqual({today: roundOf("season-0/day-5"), battle: undefined, scores: SCORES})

        hearRound(roundOf("entry-1"))
        hearRound(roundOf("season-0/day-5", 1_000))
        expect(race()).toEqual({today: undefined, battle: roundOf("entry-1"), scores: SCORES})
    })

    it("keeps the book's points through new rounds, and none of the rounds of before", () => {
        const {follow, hearScores, hearRound} = useRace.getState()
        follow("season-0", "season-0/day-5", undefined)
        hearScores(SCORES)
        hearRound(roundOf("season-0/day-5"))

        follow("season-0", "season-0/day-6", undefined)

        expect(race()).toEqual({today: undefined, battle: undefined, scores: SCORES})
    })

    it("starts again with another book", () => {
        const {follow, hearScores} = useRace.getState()
        follow("season-0", undefined, undefined)
        hearScores(SCORES)

        follow("season-1", undefined, undefined)

        expect(race()).toBeUndefined()
    })

    it("shows no race with no book, and the book's points again when it comes back", () => {
        const {follow, hearScores, leave} = useRace.getState()
        follow("season-0", undefined, undefined)
        hearScores(SCORES)

        leave()
        expect(race()).toBeUndefined()

        follow("season-0", undefined, undefined)
        expect(race()?.scores).toEqual(SCORES)
    })

    it("hears nothing before it follows a book", () => {
        useRace.getState().hearScores(SCORES)

        expect(race()).toBeUndefined()
    })
})
