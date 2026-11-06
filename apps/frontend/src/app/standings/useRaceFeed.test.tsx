// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import {act, cleanup, render} from "@testing-library/react"
import {StrictMode} from "react"
import {RaceListener, Round, Score, StandingsBackend} from "../../backends/standings.ts"
import {useRace} from "./raceStore.ts"
import {useRaceFeed} from "./useRaceFeed.ts"

const SCORES: Score[] = [{rank: 1, countryCode: "fr", points: 43, roundsWon: 1}]

const roundOf = (roundId: string): Round =>
    ({roundId, times: 1, startedAt: 0, endedAt: 0, standings: [], takes: [], players: [], before: [], after: [], start: []})

type Ids = {bookId?: string, today?: string, battle?: string}

function Harness({backend, bookId, today, battle}: Ids & {backend: StandingsBackend}) {
    useRaceFeed(backend, bookId, today, battle)
    return null
}

function streaming() {
    const streams: {roundIds: readonly string[], listener: RaceListener, stop: () => void}[] = []
    const listenForRace = vi.fn((_bookId: string, roundIds: readonly string[], listener: RaceListener) => {
        const stop = vi.fn()
        streams.push({roundIds, listener, stop})
        return stop
    })
    const backend = {listenForRace} as unknown as StandingsBackend
    const last = () => streams[streams.length - 1]
    return {backend, listenForRace, streams, last}
}

const race = () => useRace.getState().race

afterEach(() => {
    cleanup()
    useRace.setState({race: undefined, heard: undefined})
})

describe("useRaceFeed", () => {
    it("follows the book and the rounds asked into the store", () => {
        const {backend, listenForRace, last} = streaming()
        render(<Harness backend={backend} bookId="season-0" today="season-0/day-5" battle="entry-1"/>)

        expect(listenForRace).toHaveBeenCalledWith("season-0", ["season-0/day-5", "entry-1"], expect.anything())
        act(() => last().listener.onScores(SCORES))
        act(() => last().listener.onRound(roundOf("entry-1")))

        expect(race()).toEqual({today: undefined, battle: roundOf("entry-1"), scores: SCORES})
    })

    it("follows the rounds again when one changes, keeping the points, and hears nothing more of the old stream", () => {
        const {backend, streams, last} = streaming()
        const view = render(<Harness backend={backend} bookId="season-0" today="season-0/day-5"/>)
        const first = last()
        act(() => first.listener.onScores(SCORES))
        act(() => first.listener.onRound(roundOf("season-0/day-5")))

        view.rerender(<Harness backend={backend} bookId="season-0" today="season-0/day-6"/>)
        act(() => first.listener.onRound(roundOf("season-0/day-6")))

        expect(first.stop).toHaveBeenCalledTimes(1)
        expect(streams).toHaveLength(2)
        expect(race()).toEqual({today: undefined, battle: undefined, scores: SCORES})
    })

    it("shows no race with no book", () => {
        const {backend, listenForRace} = streaming()
        render(<Harness backend={backend}/>)

        expect(listenForRace).not.toHaveBeenCalled()
        expect(race()).toBeUndefined()
    })

    it("follows one stream in StrictMode", () => {
        const {backend, streams, last} = streaming()
        render(<StrictMode><Harness backend={backend} bookId="season-0"/></StrictMode>)

        expect(streams.filter((stream) => vi.mocked(stream.stop).mock.calls.length === 0)).toHaveLength(1)
        act(() => last().listener.onScores(SCORES))
        expect(race()?.scores).toEqual(SCORES)
    })

    it("stops and tells nothing once it unmounts", () => {
        const {backend, last} = streaming()
        const view = render(<Harness backend={backend} bookId="season-0"/>)
        act(() => last().listener.onScores(SCORES))

        view.unmount()

        expect(last().stop).toHaveBeenCalledTimes(1)
        expect(race()).toBeUndefined()
        expect(useRace.getState().heard).toBeUndefined()
    })
})
