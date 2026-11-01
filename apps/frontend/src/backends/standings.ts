import {NameColor, PlayerTitle} from "./player.ts"

export type Standing = {
    rank: number
    accountId: string
    name: string
    color: NameColor
    countryCode: string
    tiles: number
    wornTitle?: PlayerTitle
}

export type MySeason = {
    countryCode?: string
    tiles: number
    rank?: number
    wornTitle?: PlayerTitle
}

export type RoundStanding = {
    rank: number
    countryCode: string
    share: number
    points: number
}

export type Score = {
    rank: number
    countryCode: string
    points: number
    roundsWon: number
}

export type RoundTakes = {
    countryCode: string
    tiles: number
}

// A round of scoring, by the id the gamemaster gave it: its live standings while open, its results once scored.
export type Round = {
    roundId: string
    times: number
    startedAt: number
    // 0 while it runs.
    endedAt: number
    standings: RoundStanding[]
    takes: RoundTakes[]
    players: Standing[]
    // The book's scores before and after the round; none while it runs.
    before: Score[]
    after: Score[]
}

// The book's points, and what today's round and the running battle's would score now.
export type Race = {
    today?: Round
    battle?: Round
    scores: Score[]
}

export type Moment = "day" | "battle" | "season"

// A moment the gamemaster debriefed, with scoring's numbers.
export type ClosedRound = {
    moment: Moment
    season: number
    // A day's place among the season's days, a battle's among its battles, or the season's finale's; 0 for a season that
    // had no finale.
    number: number
    roundId: string
    endedAt: number
    // The battle that ended the season, or the season itself.
    finale: boolean
    standings: RoundStanding[]
    before: Score[]
    after: Score[]
    times: number
    startedAt: number
    // A battle's schedule entry, as the gamemaster's turn names it; empty for a day, a season, and a season's finale, which
    // is no entry's.
    entryId: string
    takes: RoundTakes[]
    players: Standing[]
}

// A scored round of a book: every country it ranked, with its share of the map, and the book's table after it.
export type BookStep = {
    roundId: string
    endedAt: number
    results: RoundStanding[]
    table: Score[]
}

export type RaceListener = {
    onScores: (scores: Score[]) => void
    onRound: (round: Round) => void
}

// The boards of one book.
export interface Boards {
    listenForStandings(countryCode: string, onStandings: (standings: Standing[]) => void): () => void

    mySeason(countryCode: string): Promise<MySeason | undefined>
}

export interface StandingsBackend {
    listenForStandings(bookId: string, countryCode: string, onStandings: (standings: Standing[]) => void): () => void

    // The book's scores and the rounds asked for, then each change.
    listenForRace(bookId: string, roundIds: readonly string[], listener: RaceListener): () => void

    mySeason(bookId: string, countryCode: string): Promise<MySeason | undefined>

    // The book's top 10 players, of every flag.
    board(bookId: string): Promise<Standing[]>

    // Each scored round of the book, by end.
    bookHistory(bookId: string): Promise<BookStep[]>

    // None for a round never opened.
    round(roundId: string): Promise<Round | undefined>

    scores(bookId: string): Promise<Score[]>
}

export function boardsOf(backend: StandingsBackend, bookId: string): Boards {
    return {
        listenForStandings: (countryCode, onStandings) => backend.listenForStandings(bookId, countryCode, onStandings),
        mySeason: (countryCode) => backend.mySeason(bookId, countryCode),
    }
}
