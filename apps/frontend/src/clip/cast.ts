import {Cast as CastWritten} from "../gen/grpc/clips/v1/clips_pb.ts"
import {Cast} from "../domain/clip/cast.ts"

// The run's cast, as the backend's clips fetch wrote it.
export function castOf(written: CastWritten): Cast {
    const names = new Map(written.fighters.map(({number, name}) => [number, name]))
    const takes = written.takes.map(({at, tileId, countryId, fighter}) => {
        if (at === undefined) throw new Error(`the cast has a take of tile ${tileId} with no time`)
        if (!names.has(fighter)) throw new Error(`the cast has a take of fighter ${fighter}, whom it does not name`)
        return {at: at.toDate().getTime(), tile: tileId, flag: countryId, fighter}
    })
    return {id: written.id, names, takes}
}
