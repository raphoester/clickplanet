export type Season = {
    number: number
    finaleStartsAt: number
    endsAt: number
    startsAt?: number
}

export interface SeasonBackend {
    season(): Promise<Season | undefined>

    next(): Promise<Season | undefined>
}
