export type Capitals = ReadonlyMap<number, string>

// Each country's capital, as the map generator wrote it: {"tiles": N, "capitals": {"fr": tile, …}}.
export function capitalsOf(json: unknown, tiles: number): Capitals {
    if (!isRecord(json) || json.tiles !== tiles || !isRecord(json.capitals)) {
        throw new Error(`the capitals are not for a map of ${tiles} tiles`)
    }
    const capitals = new Map<number, string>()
    for (const [code, tile] of Object.entries(json.capitals)) {
        if (!isTile(tile, tiles)) throw new Error(`the capital of ${code} is not a tile: ${String(tile)}`)
        capitals.set(tile, code)
    }
    return capitals
}

export async function loadCapitals(url: string, tiles: number, signal?: AbortSignal): Promise<Capitals> {
    const response = await fetch(url, {signal})
    if (!response.ok) throw new Error(`${url} answered ${response.status}`)
    return capitalsOf(await response.json(), tiles)
}

export function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === "object" && value !== null && !Array.isArray(value)
}

export function isTile(value: unknown, tiles: number): value is number {
    return Number.isInteger(value) && (value as number) >= 1 && (value as number) <= tiles
}
