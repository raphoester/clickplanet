export type Season = {
    number: number
    finaleStartsAt: number
    endsAt: number
}

export interface SeasonBackend {
    season(): Promise<Season | undefined>
}
