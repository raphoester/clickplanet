export type Season = {
    number: number
    // When the season's last day ends: every day ends at this time of day.
    lastDayEndsAt: number
    endsAt: number
}

export interface SeasonBackend {
    season(): Promise<Season | undefined>
}
