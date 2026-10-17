export type Allegiance = {
    countryId: string

    // Milliseconds since the epoch; undefined for a flag never changed.
    changeableAt?: number
}

export function lockOf(allegiance: Allegiance | undefined, now: number): Allegiance | undefined {
    return allegiance?.changeableAt !== undefined && allegiance.changeableAt > now ? allegiance : undefined
}
