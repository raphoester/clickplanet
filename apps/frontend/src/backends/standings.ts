import {NameColor} from "./player.ts"

export type Standing = {
    rank: number
    name: string
    color: NameColor
    countryCode: string
    tiles: number
}

export type MySeason = {
    countryCode?: string
    tiles: number
    globalRank?: number
    countryRank?: number
}

export interface StandingsBackend {
    standings(countryCode: string): Promise<Standing[]>

    mySeason(): Promise<MySeason | undefined>
}
