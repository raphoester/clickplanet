import {type Bridge, Bridges} from "../../domain/bridges.ts"
import {isRecord, isTile} from "./capitals.ts"

// Each crossing's two end tiles, as the map generator wrote them: {"tiles": N, "bridges": [[tile, tile], …]}.
export function bridgesOf(json: unknown, tiles: number): Bridges {
    if (!isRecord(json) || json.tiles !== tiles || !Array.isArray(json.bridges)) {
        throw new Error(`the bridges are not for a map of ${tiles} tiles`)
    }
    const all = json.bridges.map((pair: unknown, i): Bridge => {
        if (!Array.isArray(pair) || pair.length !== 2 || !pair.every((tile) => isTile(tile, tiles))) {
            throw new Error(`bridge ${i} is not two tiles`)
        }
        return [pair[0], pair[1]]
    })
    return new Bridges(all)
}

export async function loadBridges(url: string, tiles: number, signal?: AbortSignal): Promise<Bridges> {
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    return bridgesOf(await response.json(), tiles)
}
