// A season, as the gamemaster's timeline names it.
export type Season = {
    number: number
    startsAt: number
    endsAt: number
    // Its book in scoring: the board, the points and the tally are read by it.
    bookId: string
}

// The day in progress: a round scored at the daily cutoff.
export type Day = {
    number: number
    startsAt: number
    endsAt: number
    roundId: string
}

export type Seasons = {
    // The season that runs now; none between two seasons.
    current?: Season
    next?: Season
    // None between two seasons, and from the season's last cutoff to its end.
    today?: Day
    // By number.
    ended: Season[]
}

export const NO_SEASONS: Seasons = {ended: []}
