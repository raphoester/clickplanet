import {create} from "zustand"
import {Race, Round, Score} from "../../backends/standings.ts"
import {raceOf} from "../../domain/race.ts"

// What the stream told of the book, for the rounds asked.
type Heard = {
    bookId: string
    key: string
    todayRoundId?: string
    battleRoundId?: string
    scores?: Score[]
    rounds: ReadonlyMap<string, Round>
}

type RaceStore = {
    race?: Race
    heard?: Heard
    follow: (bookId: string, todayRoundId: string | undefined, battleRoundId: string | undefined) => void
    hearScores: (scores: Score[]) => void
    hearRound: (round: Round) => void
    leave: () => void
}

const NO_ROUNDS: ReadonlyMap<string, Round> = new Map()

const keyOf = (todayRoundId: string | undefined, battleRoundId: string | undefined) => `${todayRoundId ?? ""} ${battleRoundId ?? ""}`

const told = (heard: Heard) => ({heard, race: raceOf(heard.scores, heard.rounds, heard.todayRoundId, heard.battleRoundId)})

export const useRace = create<RaceStore>()((set) => ({
    // The book's points stay through new rounds; a round only while the same ones are asked.
    follow: (bookId, todayRoundId, battleRoundId) => set(({heard}) => {
        const key = keyOf(todayRoundId, battleRoundId)
        const book = heard?.bookId === bookId ? heard : undefined
        return told({
            bookId,
            key,
            todayRoundId,
            battleRoundId,
            scores: book?.scores,
            rounds: book?.key === key ? book.rounds : NO_ROUNDS,
        })
    }),
    hearScores: (scores) => set((race) => race.heard ? told({...race.heard, scores}) : race),
    hearRound: (round) => set((race) => race.heard
        ? told({...race.heard, rounds: new Map(race.heard.rounds).set(round.roundId, round)})
        : race),
    // No race shows with no book; what was heard of one comes back with it.
    leave: () => set({race: undefined}),
}))
