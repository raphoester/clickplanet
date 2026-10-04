import {Title as TitlePb} from "../gen/grpc/player/v1/title_pb.ts"

export type TitleRank = {
    trackId: string
    trackName: string
    number: number
    count: number
}

export type PlayerTitle = {
    id: string
    name: string
    rank?: TitleRank
}

export function titleOf(title: TitlePb | undefined): PlayerTitle | undefined {
    if (!title || !title.id) return undefined

    const rank = title.rank
    return {
        id: title.id,
        name: title.name,
        rank: rank && rank.count > 0
            ? {trackId: rank.trackId, trackName: rank.trackName, number: rank.number, count: rank.count}
            : undefined,
    }
}
