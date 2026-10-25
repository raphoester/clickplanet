import {Season, SeasonBackend} from "./season.ts"

export const SEASON_ZERO: Season = {
    number: 0,
    finaleStartsAt: Date.UTC(2026, 9, 31, 21),
    endsAt: Date.UTC(2026, 9, 31, 23),
}

export const SEASON_ONE: Season = {
    number: 1,
    startsAt: Date.UTC(2026, 10, 1, 19),
    finaleStartsAt: Date.UTC(2026, 11, 5, 21),
    endsAt: Date.UTC(2026, 11, 5, 23),
}

export class FakeSeasonBackend implements SeasonBackend {
    constructor(private readonly current: Season | undefined, private readonly coming?: Season) {
    }

    public async season(): Promise<Season | undefined> {
        return this.current
    }

    public async next(): Promise<Season | undefined> {
        return this.coming
    }
}
