import {useEffect} from "react"
import {StandingsBackend} from "../../backends/standings.ts"
import {useRace} from "./raceStore.ts"

// Follows the book's points, and today's round and the running battle's while they are open, into the store.
export function useRaceFeed(
    backend: StandingsBackend | undefined,
    bookId: string | undefined,
    todayRoundId: string | undefined,
    battleRoundId: string | undefined,
): void {
    useEffect(() => {
        const {follow, hearScores, hearRound, leave} = useRace.getState()
        if (!backend || !bookId) {
            leave()
            return
        }

        let stale = false
        follow(bookId, todayRoundId, battleRoundId)
        const roundIds = [todayRoundId, battleRoundId].filter((roundId): roundId is string => !!roundId)
        const stop = backend.listenForRace(bookId, roundIds, {
            onScores: (scores) => {
                if (!stale) hearScores(scores)
            },
            onRound: (round) => {
                if (!stale) hearRound(round)
            },
        })
        return () => {
            stale = true
            stop()
        }
    }, [backend, bookId, todayRoundId, battleRoundId])

    // Nothing is told once nothing follows the race.
    useEffect(() => () => useRace.setState({race: undefined, heard: undefined}), [])
}
