export type Season = {
    number: number
    finaleStartsAt: number
    endsAt: number
    finaleFile?: string
}

export interface SeasonBackend {
    season(): Promise<Season | undefined>
}
