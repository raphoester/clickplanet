import {spacingOf} from "./spacing.mjs"
import {SEA, mostlyLake} from "./ground.mjs"
import {lonLatOf} from "./lattice.mjs"
import {SEATS, seat} from "./seats.mjs"

/**
 * The oracle's answer for every lattice vertex: Natural Earth's country, sea where most of the cell is lake, then
 * each seat. `npm run map:generate` writes the board from it and `npm run map:audit` checks the board against it.
 * @param {{groundOf: (lon: number, lat: number) => string, lakeAt: (lon: number, lat: number) => boolean}} ground
 * @param {{count: number, positions: Float32Array}} vertices the whole lattice
 * @param {{count: number, at: Uint32Array, to: Uint32Array}} edges the lattice's, from `neighbours`
 * @returns {{grounds: string[], lakes: number, seated: {code: string, vertex: number, was: string}[]}}
 */
export function latticeGround(ground, {count, positions}, edges, seats = SEATS) {
    const spacing = spacingOf(positions, edges)
    const grounds = new Array(count)
    let lakes = 0
    for (let v = 0; v < count; v++) {
        const [lon, lat] = lonLatOf(positions, v)
        let code = ground.groundOf(lon, lat)
        if (code !== SEA && mostlyLake(ground, lon, lat, spacing)) {
            code = SEA
            lakes++
        }
        grounds[v] = code
    }
    return {grounds, lakes, seated: seat(grounds, positions, edges, seats)}
}
