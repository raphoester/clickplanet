import {MapFrame, MapHistory} from "../domain/mapHistory.ts"

// A made-up history ending on the map given: each frame, a few hundred tiles go to the flag of a tile near them.
export function fakeHistoryOf(final: ReadonlyMap<number, string>, since: number, until: number, count: number, random = Math.random): MapHistory {
    const tiles = [...final.keys()]
    const owners = new Map(final)
    const backwards: {tile: number, owner: string}[][] = []
    const hits = new Map<number, number>()
    const hot = tiles.filter(() => random() < 0.001)

    for (let i = 1; i < count; i++) {
        const changes: {tile: number, owner: string}[] = []
        for (let j = 0; j < 300; j++) {
            const center = hot[Math.floor(random() * hot.length)] ?? tiles[0]
            const tile = center + Math.floor(random() * 120) - 60
            const other = tiles[Math.floor(random() * tiles.length)]
            if (!owners.has(tile)) continue
            changes.push({tile, owner: owners.get(tile) ?? ""})
            owners.set(tile, owners.get(other) ?? "")
            hits.set(tile, (hits.get(tile) ?? 0) + 1 + Math.floor(random() * 20))
        }
        backwards.push(changes)
    }

    const frames: MapFrame[] = [frameOf(since, [...owners].map(([tile, owner]) => ({tile, owner})))]
    backwards.reverse().forEach((changes, i) => {
        frames.push(frameOf(since + ((i + 1) * (until - since)) / (count - 1), changes.reverse()))
    })
    return {frames, hits}
}

function frameOf(at: number, changes: {tile: number, owner: string}[]): MapFrame {
    const last = new Map(changes.map(({tile, owner}) => [tile, owner]))
    return {at, tiles: Uint32Array.from(last.keys()), owners: [...last.values()]}
}
