import {Season, SeasonBackend} from "./season.ts"

export const SEASON_ZERO: Season = {
    number: 0,
    finaleStartsAt: Date.UTC(2026, 9, 31, 21),
    endsAt: Date.UTC(2026, 9, 31, 23),
}

export class FakeSeasonBackend implements SeasonBackend {
    constructor(private readonly current: Season | undefined) {
    }

    public async season(): Promise<Season | undefined> {
        return this.current
    }
}
