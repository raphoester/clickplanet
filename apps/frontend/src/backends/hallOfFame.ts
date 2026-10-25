import {Score, Standing} from "./standings.ts"

export type EndedSeason = {
    number: number
    endedAt: number
    scores: Score[]
    standings: Standing[]
}

export interface HallOfFameBackend {
    hallOfFame(): Promise<EndedSeason[]>
}
