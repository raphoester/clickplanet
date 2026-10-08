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

export type RoundStanding = {
    rank: number
    countryCode: string
    share: number
    points: number
}

export type Round = {
    number: number
    endsAt: number
    finale: boolean
    standings: RoundStanding[]
}

export type Score = {
    rank: number
    countryCode: string
    points: number
    roundsWon: number
}

export type Race = {
    round?: Round
    scores: Score[]
}

export interface StandingsBackend {
    listenForStandings(countryCode: string, onStandings: (standings: Standing[]) => void): () => void

    listenForRace(onRace: (race: Race) => void): () => void

    mySeason(countryCode: string): Promise<MySeason | undefined>
}
