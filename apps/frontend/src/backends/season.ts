// A season, as the gamemaster's timeline names it.
export type Season = {
    number: number
    startsAt: number
    endsAt: number
}

export type Seasons = {
    // The season that runs now; none between two seasons.
    current?: Season
    next?: Season
}
