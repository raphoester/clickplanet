import {useEffect, useState} from "react"
import {Race, Round, Score, StandingsBackend} from "../../backends/standings.ts"
import {raceOf} from "../../domain/race.ts"

type Read = {
    bookId: string
    key: string
    scores?: Score[]
    rounds: ReadonlyMap<string, Round>
}

const NO_ROUNDS: ReadonlyMap<string, Round> = new Map()

// The book's points, and today's round and the running battle's while they are open.
export function useRace(
    backend: StandingsBackend | undefined,
    bookId: string | undefined,
    todayRoundId: string | undefined,
    battleRoundId: string | undefined,
): Race | undefined {
    const [read, setRead] = useState<Read>()
    const key = `${todayRoundId ?? ""} ${battleRoundId ?? ""}`

    useEffect(() => {
        if (!backend || !bookId) return

        let stale = false
        const kept = (was: Read | undefined): Read => ({
            bookId,
            key,
            scores: was?.bookId === bookId ? was.scores : undefined,
            rounds: was?.bookId === bookId && was.key === key ? was.rounds : NO_ROUNDS,
        })
        const roundIds = [todayRoundId, battleRoundId].filter((roundId): roundId is string => !!roundId)
        const stop = backend.listenForRace(bookId, roundIds, {
            onScores: (scores) => {
                if (!stale) setRead((was) => ({...kept(was), scores}))
            },
            onRound: (round) => {
                if (!stale) setRead((was) => {
                    const now = kept(was)
                    return {...now, rounds: new Map(now.rounds).set(round.roundId, round)}
                })
            },
        })
        return () => {
            stale = true
            stop()
        }
    }, [backend, bookId, todayRoundId, battleRoundId, key])

    if (!backend || !bookId || read?.bookId !== bookId) return undefined
    return raceOf(read.scores, read.key === key ? read.rounds : NO_ROUNDS, todayRoundId, battleRoundId)
}
