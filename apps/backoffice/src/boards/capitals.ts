import {boardOf} from "./boards.ts"
import {loadPlaces, Places} from "./places.ts"

// A country's capital, where its tile is on a board, in degrees.
export type Capital = {country: string, latitude: number, longitude: number}

// A board's capitals file (/map/README.md#capitals-and-bridges): each country's capital as a tile id of that board.
type CapitalsFile = {tiles: number, capitals: Record<string, number>}

// Each capital at its tile. A file of another board's tiles, or a tile the board does not hold, is refused.
export function capitalsOf(places: Places, file: string): Capital[] {
    const {tiles, capitals} = JSON.parse(file) as CapitalsFile
    const {name} = places.board
    const count = places.latitudes.length
    if (tiles !== count) throw new Error(`the capitals are of a board of ${tiles} tiles, ${name} has ${count}`)
    return Object.entries(capitals).map(([country, tile]) => {
        if (!Number.isInteger(tile) || tile < 1 || tile > count) throw new Error(`${country}'s capital is tile ${tile}, not one of ${name}`)
        return {country, latitude: places.latitudes[tile - 1], longitude: places.longitudes[tile - 1]}
    })
}

// A board's capitals, placed once its places are read.
export async function loadCapitals(name: string): Promise<Capital[]> {
    return capitalsOf(await loadPlaces(name), boardOf(name).capitals)
}
