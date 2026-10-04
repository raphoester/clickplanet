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
    standings(countryCode: string): Promise<Standing[]>

    mySeason(countryCode: string): Promise<MySeason | undefined>
}
