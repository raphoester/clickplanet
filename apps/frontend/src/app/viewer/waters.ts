import {Waters} from "../../domain/waters.ts"
import {isRecord, isTile} from "./capitals.ts"

// Each body of water's coast, as the map generator wrote it: {"tiles": N, "bodies": [[tile, …], …]}.
export function watersOf(json: unknown, tiles: number): Waters {
    if (!isRecord(json) || json.tiles !== tiles || !Array.isArray(json.bodies)) {
        throw new Error(`the waters are not for a map of ${tiles} tiles`)
    }
    const bodies = json.bodies.map((coast: unknown, body) => {
        if (!Array.isArray(coast) || !coast.every((tile) => isTile(tile, tiles))) {
            throw new Error(`the coast of water ${body} is not a list of tiles`)
        }
        return coast as number[]
    })
    return new Waters(bodies)
}

export async function loadWaters(url: string, tiles: number, signal?: AbortSignal): Promise<Waters> {
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    return watersOf(await response.json(), tiles)
}
