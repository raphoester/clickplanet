import {PlayerTitle} from "./title.ts"

export type {PlayerTitle, TitleRank} from "./title.ts"

export type TitleStep = {
    title: PlayerTitle
    threshold: number
    earned: boolean
}

export type TitleTrack = {
    id: string
    name: string
    progress: number
    steps: TitleStep[]
}

export type TitleDashboard = {
    worn?: PlayerTitle
    wearable: PlayerTitle[]
    tracks: TitleTrack[]
}

export type Fronts = {
    playsFor: CountryTiles[]
    playsAgainst: CountryTiles[]
}

export type CountryTiles = {
    countryCode: string
    tiles: number
}

export interface ProgressionBackend {
    titles(): Promise<TitleDashboard>

    wearTitle(id: string): Promise<PlayerTitle | undefined>

    fronts(): Promise<Fronts>

    // The caller's own level, none for a guest or an account no fact has raised yet.
    level(): Promise<PlayerLevel | undefined>
}

// The level kept, on its bar: the XP can pass nextLevelXp until the raise lands.
export type PlayerLevel = {
    level: number
    xp: number
    levelXp: number
    nextLevelXp: number
}

export type Progress = {
    tilesTaken: number
    streakCurrent: number
    streakBest: number
    titles: PlayerTitle[]
    wornTitle?: PlayerTitle
    level?: PlayerLevel
} & Fronts

export const NO_PROGRESS: Progress = {
    tilesTaken: 0, streakCurrent: 0, streakBest: 0, titles: [], playsFor: [], playsAgainst: [],
}

export interface ProgressReader {
    // Undefined for an account the other players do not see: a guest, one asked deleted, one never named.
    progress(accountId: string): Promise<Progress | undefined>
}

export type ProgressionEvents = {
    titleEarned: (title: PlayerTitle) => void
    levelReached: (level: PlayerLevel) => void
}

export interface ProgressionListener {
    heldIdentity(): string | undefined

    // What the account of the token held earns. A refused token stops it for good: a new token needs a new one.
    listenForProgression(on: ProgressionEvents): () => void
}
