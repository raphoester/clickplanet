import {Board} from "../boards/boards.ts"
import {Places} from "../boards/places.ts"
import {EARTH_1} from "../boards/testBoards.ts"

// For the tests: a patch of a board's lattice, a hexagon of tiles `radius` steps wide around a place, a step of the
// board's spacing (0.23° on earth-1). Each tile's id is its index plus one; `idOf(i, j)` finds the tile `i` steps east
// and `j` rows north (each row half a step east of the one below).
export function patch(latitude: number, longitude: number, radius = 12, board: Board = EARTH_1) {
    const step = board.spacing
    const lat = latitude * Math.PI / 180, lon = longitude * Math.PI / 180
    const centre = [-Math.cos(lat) * Math.cos(lon), Math.sin(lat), Math.cos(lat) * Math.sin(lon)]
    const east = [Math.sin(lon), 0, Math.cos(lon)]
    const north = [Math.sin(lat) * Math.cos(lon), Math.cos(lat), -Math.sin(lat) * Math.sin(lon)]
    const latitudes: number[] = [], longitudes: number[] = []
    const ids = new Map<string, number>()
    for (let j = -radius; j <= radius; j++) {
        for (let i = -radius; i <= radius; i++) {
            if (Math.abs(i + j) > radius) continue
            const x = (i + j / 2) * step, y = j * step * Math.sqrt(3) / 2
            const p = [0, 1, 2].map(k => centre[k] + x * east[k] + y * north[k])
            const length = Math.hypot(p[0], p[1], p[2])
            latitudes.push(Math.asin(p[1] / length) * 180 / Math.PI)
            longitudes.push(Math.atan2(p[2], -p[0]) * 180 / Math.PI)
            ids.set(`${i},${j}`, latitudes.length)
        }
    }
    const places: Places = {board, latitudes: Float32Array.from(latitudes), longitudes: Float32Array.from(longitudes)}
    return {places, idOf: (i: number, j: number) => ids.get(`${i},${j}`)!}
}
