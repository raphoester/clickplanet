import {latticeTriangles} from "../src/domain/lattice.ts"
import {boardAsked, boardDir, readBoard} from "./map/board.mjs"
import {lattice, neighbours} from "./map/lattice.mjs"
import {cellAreas} from "./map/warp.mjs"

// `npm run map:measure -- earth-2`: what docs/boards.md records of a board's geometry. Read-only.
//
// - the spread of the cells' areas, largest over smallest, the twelve pentagons left out;
// - the closest two touching tiles over the median spacing, on land;
// - the tiles a 4-ring bomb takes wherever it lands, and on a pentagon: every point within 4 × the mean arc between
//   touching tiles (the backend's `Geography.Spacing`, which a bomb's radius reads) of the point hit, counted on the
//   whole lattice so the sea does not cut the circle;
// - each pentagon's distance to the nearest tile;
// - the creases: how far a row of cells turns at a point, the angle between a neighbour and the one opposite it, at a
//   point on one of the 30 face edges against one inside a face, on land.

const EARTH_KM = 6371
const BOMB_RINGS = 4

const board = readBoard(boardDir(boardAsked(process.argv.slice(2)).name))
const shape = board.manifest.lattice
const {cols} = shape
const {count: points, positions} = lattice(shape)
const {at, to} = neighbours(shape)
const tileOf = new Int32Array(points).fill(-1)
for (let t = 0; t < board.count; t++) tileOf[board.points[t]] = t
const idOf = new Uint32Array(points)
board.points.forEach((point, t) => {
    idOf[point] = board.ids[t]
})
// The canonical id of every point, in the lattice's order.
const {latticeIds} = await import("../src/domain/lattice.ts")
const ids = latticeIds(shape)

const arc = (a: number, b: number) => 2 * Math.asin(Math.min(1, Math.hypot(
    positions[a * 3] - positions[b * 3], positions[a * 3 + 1] - positions[b * 3 + 1], positions[a * 3 + 2] - positions[b * 3 + 2],
) / 2))
const isCorner = (point: number) => ids[point] < 12
const onFaceEdge = (point: number) => ids[point] >= 12 && ids[point] < 12 + 30 * (cols - 1)

// The cells' areas.
const areas = cellAreas(shape, latticeTriangles(shape))
let leastAll = Infinity, mostAll = 0, leastLand = Infinity, mostLand = 0
for (let point = 0; point < points; point++) {
    if (isCorner(point)) continue
    leastAll = Math.min(leastAll, areas[point])
    mostAll = Math.max(mostAll, areas[point])
    if (tileOf[point] < 0) continue
    leastLand = Math.min(leastLand, areas[point])
    mostLand = Math.max(mostLand, areas[point])
}

// The touching tiles: the closest, and the mean the backend's spacing is.
let closest = Infinity, sum = 0, edges = 0
for (let point = 0; point < points; point++) {
    if (tileOf[point] < 0) continue
    for (let e = at[point]; e < at[point + 1]; e++) {
        const other = to[e]
        if (other < point || tileOf[other] < 0) continue
        const length = arc(point, other)
        closest = Math.min(closest, length)
        sum += length
        edges++
    }
}
const meanSpacing = sum / edges

// A 4-ring bomb: the points within its radius, found by a walk out from the point hit.
const radius = BOMB_RINGS * meanSpacing
const seen = new Int32Array(points).fill(-1)
const bombAt = (centre: number): number => {
    let taken = 0
    const stack = [centre]
    seen[centre] = centre
    while (stack.length > 0) {
        const point = stack.pop()!
        taken++
        for (let e = at[point]; e < at[point + 1]; e++) {
            const other = to[e]
            if (seen[other] === centre || arc(centre, other) > radius) continue
            seen[other] = centre
            stack.push(other)
        }
    }
    return taken
}
let fewest = Infinity, most = 0
const bombs = new Map<number, number>()
for (let point = 0; point < points; point++) {
    const taken = bombAt(point)
    bombs.set(taken, (bombs.get(taken) ?? 0) + 1)
    if (isCorner(point)) continue
    fewest = Math.min(fewest, taken)
    most = Math.max(most, taken)
}
const pentagons: {point: number, bomb: number, landKm: number, lat: number, lon: number}[] = []
for (let point = 0; point < points; point++) {
    if (!isCorner(point)) continue
    let nearest = Infinity
    for (let t = 0; t < board.count; t++) nearest = Math.min(nearest, arc(point, board.points[t]))
    const x = positions[point * 3], y = positions[point * 3 + 1], z = positions[point * 3 + 2]
    pentagons.push({
        point, bomb: bombAt(point), landKm: nearest * EARTH_KM,
        lat: Math.asin(y) * 180 / Math.PI, lon: Math.atan2(z, -x) * 180 / Math.PI,
    })
}

// The creases: at each point of six neighbours, each neighbour against the one opposite it around the point.
const turns = {edge: [] as number[], inside: [] as number[]}
for (let point = 0; point < points; point++) {
    if (tileOf[point] < 0 || at[point + 1] - at[point] !== 6) continue
    const px = positions[point * 3], py = positions[point * 3 + 1], pz = positions[point * 3 + 2]
    // A frame of the tangent plane.
    const side = Math.abs(px) < 0.9 ? [1, 0, 0] : [0, 1, 0]
    let ux = py * side[2] - pz * side[1], uy = pz * side[0] - px * side[2], uz = px * side[1] - py * side[0]
    const ul = Math.hypot(ux, uy, uz)
    ux /= ul; uy /= ul; uz /= ul
    const vx = py * uz - pz * uy, vy = pz * ux - px * uz, vz = px * uy - py * ux
    const around: {angle: number, x: number, y: number}[] = []
    for (let e = at[point]; e < at[point + 1]; e++) {
        const other = to[e]
        const dx = positions[other * 3] - px, dy = positions[other * 3 + 1] - py, dz = positions[other * 3 + 2] - pz
        const x = dx * ux + dy * uy + dz * uz, y = dx * vx + dy * vy + dz * vz
        around.push({angle: Math.atan2(y, x), x, y})
    }
    around.sort((a, b) => a.angle - b.angle)
    let worst = 0
    for (let k = 0; k < 3; k++) {
        const a = around[k], b = around[k + 3]
        const straight = Math.acos(Math.max(-1, Math.min(1, (a.x * b.x + a.y * b.y) / (Math.hypot(a.x, a.y) * Math.hypot(b.x, b.y)))))
        worst = Math.max(worst, 180 - straight * 180 / Math.PI)
    }
    ;(onFaceEdge(point) ? turns.edge : turns.inside).push(worst)
}
const quantile = (values: number[], q: number) => Float64Array.from(values).sort()[Math.min(values.length - 1, Math.floor(values.length * q))]

const km = (radians: number) => (radians * EARTH_KM).toFixed(1)
console.log(`${board.manifest.name}: cols ${cols}, ${board.count} tiles of ${points} points, median spacing ${km(board.manifest.spacing)} km, mean ${km(meanSpacing)} km`)
console.log(`cell area, largest ÷ smallest: ${(mostAll / leastAll).toFixed(3)} on the lattice, ${(mostLand / leastLand).toFixed(3)} on land (the pentagons left out)`)
console.log(`closest two touching tiles ÷ median spacing, on land: ${(closest / board.manifest.spacing).toFixed(3)} (${km(closest)} km)`)
console.log(`a ${BOMB_RINGS}-ring bomb (radius ${km(radius)} km) takes ${fewest} to ${most} points wherever it lands; on a pentagon ${pentagons.map(({bomb}) => bomb).join(", ")}`)
console.log(`  how many points take each count: ${[...bombs.entries()].sort((a, b) => a[0] - b[0]).map(([n, c]) => `${n}: ${c}`).join(", ")}`)
console.log("the pentagons, nearest land:")
for (const {lat, lon, landKm} of pentagons) console.log(`  ${lat.toFixed(2)}, ${lon.toFixed(2)}: ${landKm.toFixed(0)} km`)
console.log(`a row's turn at a point on land, degrees: on a face edge, median ${quantile(turns.edge, 0.5).toFixed(2)}, 99% ${quantile(turns.edge, 0.99).toFixed(2)}, `
    + `most ${Math.max(...turns.edge).toFixed(2)} (${turns.edge.length} points); inside a face, median ${quantile(turns.inside, 0.5).toFixed(2)}, `
    + `99% ${quantile(turns.inside, 0.99).toFixed(2)}, most ${quantile(turns.inside, 1).toFixed(2)}`)
