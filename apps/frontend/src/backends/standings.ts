import {NameColor, PlayerTitle} from "./player.ts"

export type Standing = {
    rank: number
    name: string
    color: NameColor
    countryCode: string
    tiles: number
    wornTitle?: PlayerTitle
}

export type MySeason = {
    countryCode?: string
    tiles: number
    rank?: number
    wornTitle?: PlayerTitle
}

export interface StandingsBackend {
    listenForStandings(countryCode: string, onStandings: (standings: Standing[]) => void): () => void

    mySeason(countryCode: string): Promise<MySeason | undefined>
}
