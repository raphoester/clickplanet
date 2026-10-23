export const TILE_PARAM = "tile"

// A link that opens the globe on a tile: the operator's page sends one to see a reported drawing.
export function linkedTile(url: URL): number | undefined {
    const value = url.searchParams.get(TILE_PARAM)
    if (value === null || !/^[1-9][0-9]{0,9}$/.test(value.trim())) return undefined

    const tile = Number(value.trim())
    return tile <= 0xFFFFFFFF ? tile : undefined
}

export function withoutLinkedTile(url: URL): URL {
    const stripped = new URL(url)
    stripped.searchParams.delete(TILE_PARAM)
    return stripped
}
