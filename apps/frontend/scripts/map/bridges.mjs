import {positionOf} from "./seats.mjs"

const EARTH_KM = 6371

// A lane's end lands on a coast tile this near its point, or the lane is refused: a point placed off its shore.
export const SNAP_KM = 150

/**
 * Sea lanes: crossings placed by hand over the seas the stepping stones do not cross, each a point [lon, lat] on either
 * shore. An end snaps to the nearest coast tile that is no capital. The points are pinned, like the seats: moving one
 * moves a bridge.
 * @type {Record<string, [[number, number], [number, number]]>}
 */
export const LANES = {
    "North Atlantic: Ireland – Newfoundland": [[-10.0, 53.5], [-52.7, 47.6]],
    "Columbus: Canaries – Puerto Rico": [[-15.6, 28.1], [-65.7, 18.35]],
    "Florida Straits: Key West – Cuba": [[-81.8, 24.55], [-81.6, 23.05]],
    "South Atlantic: Senegal – Brazil": [[-17.2, 14.8], [-35.2, -5.8]],
    "Cape: South Africa – Brazil": [[18.4, -33.9], [-43.2, -22.95]],
    "Gibraltar: Spain – Morocco": [[-5.6, 36.02], [-5.8, 35.78]],
    "Sicily – Tunisia": [[12.45, 37.8], [11.0, 37.05]],
    "Bab-el-Mandeb: Yemen – Djibouti": [[43.45, 12.65], [43.3, 12.45]],
    "Hormuz: Iran – Oman": [[56.3, 27.15], [56.25, 26.2]],
    "Monsoon: Kenya – India": [[39.7, -4.05], [76.25, 9.95]],
    "Indian Ocean: South Africa – Australia": [[31.0, -29.9], [115.75, -32.0]],
    "Bay of Bengal: Sri Lanka – Sumatra": [[81.25, 8.55], [95.3, 5.55]],
    "Korea Strait: Busan – Fukuoka": [[129.05, 35.1], [130.4, 33.6]],
    "Tasman: Sydney – Auckland": [[151.25, -33.85], [174.8, -36.85]],
    "Galleon: Philippines – Mexico": [[122.2, 14.1], [-99.9, 16.85]],
    "North Pacific: Japan – Hawaii": [[140.85, 35.7], [-158.1, 21.4]],
    "North Pacific: Hawaii – California": [[-157.85, 21.3], [-122.5, 37.75]],
}

/**
 * Every bridge of the map: the stepping stones, which join two pieces of land when no third piece is nearer to both
 * (so every piece is joined, and a strait or an island chain is crossed where it is narrowest), and the sea lanes.
 * A piece is land that touches, whatever its countries. A bridge lands on a coast tile, and on a capital only when its
 * piece has no other coast.
 * @param {{positions: ArrayLike<number>, at: Uint32Array, to: Uint32Array, tileOf: Int32Array,
 *   capitals: Record<string, number>}} map the tiles' positions (x, y, z per tile), the lattice's edges by vertex, each
 *   vertex's tile index or -1 at sea, and each country's capital as a wire id
 * @param {typeof LANES} lanes
 * @returns {{bridges: [number, number][], pieces: number, stones: number, lanes: number}} each bridge's ends as wire
 *   ids, the smaller first, in order
 */
export function bridgesOf({positions, at, to, tileOf, capitals}, lanes = LANES) {
    const tiles = positions.length / 3
    const {pieceOf, pieces, coast} = piecesOf(tiles, {at, to, tileOf})
    const capital = new Set(Object.values(capitals).map((id) => id - 1))

    const shores = Array.from({length: pieces}, () => [])
    for (let t = 0; t < tiles; t++) if (coast[t] && !capital.has(t)) shores[pieceOf[t]].push(t)
    for (let t = 0; t < tiles; t++) if (coast[t] && capital.has(t) && shores[pieceOf[t]].length === 0) shores[pieceOf[t]].push(t)
    const ends = shores.flat()

    const closest = new Float64Array(pieces * pieces).fill(-2)
    const from = new Int32Array(pieces * pieces)
    const onto = new Int32Array(pieces * pieces)
    for (let i = 0; i < ends.length; i++) {
        const a = ends[i]
        const pa = pieceOf[a]
        const ax = positions[a * 3], ay = positions[a * 3 + 1], az = positions[a * 3 + 2]
        for (let j = i + 1; j < ends.length; j++) {
            const b = ends[j]
            const pb = pieceOf[b]
            if (pa === pb) continue
            const along = ax * positions[b * 3] + ay * positions[b * 3 + 1] + az * positions[b * 3 + 2]
            const k = pa < pb ? pa * pieces + pb : pb * pieces + pa
            if (along > closest[k]) {
                closest[k] = along
                from[k] = a
                onto[k] = b
            }
        }
    }

    const apart = (a, b) => a === b ? -Infinity : -closest[a < b ? a * pieces + b : b * pieces + a]
    const bridges = new Map()
    const add = (a, b) => {
        const pair = a < b ? [a + 1, b + 1] : [b + 1, a + 1]
        bridges.set(`${pair[0]},${pair[1]}`, pair)
    }
    for (let a = 0; a < pieces; a++) {
        for (let b = a + 1; b < pieces; b++) {
            const gap = apart(a, b)
            let nearer = false
            for (let c = 0; c < pieces && !nearer; c++) {
                if (c !== a && c !== b && Math.max(apart(a, c), apart(b, c)) < gap) nearer = true
            }
            if (!nearer) add(from[a * pieces + b], onto[a * pieces + b])
        }
    }
    const stones = bridges.size

    const landings = []
    for (let t = 0; t < tiles; t++) if (coast[t] && !capital.has(t)) landings.push(t)
    for (const [name, [start, end]] of Object.entries(lanes)) {
        const a = snap(positions, landings, start, name)
        const b = snap(positions, landings, end, name)
        if (a === b) throw new Error(`${name}: both ends snap to tile ${a + 1}`)
        add(a, b)
    }

    const sorted = [...bridges.values()].sort((x, y) => x[0] - y[0] || x[1] - y[1])
    return {bridges: sorted, pieces, stones, lanes: Object.keys(lanes).length}
}

/**
 * The pieces of land no bridge reaches from the first tile's: none on a map whose bridges join it.
 * @param {{at: Uint32Array, to: Uint32Array, tileOf: Int32Array}} lattice
 * @param {number} tiles
 * @param {[number, number][]} bridges
 */
export function piecesApart(lattice, tiles, bridges) {
    const {pieceOf, pieces} = piecesOf(tiles, lattice)
    const across = Array.from({length: pieces}, () => [])
    for (const [a, b] of bridges) {
        across[pieceOf[a - 1]].push(pieceOf[b - 1])
        across[pieceOf[b - 1]].push(pieceOf[a - 1])
    }
    const seen = new Uint8Array(pieces)
    const queue = [pieceOf[0]]
    seen[pieceOf[0]] = 1
    while (queue.length > 0) {
        for (const next of across[queue.pop()]) {
            if (!seen[next]) {
                seen[next] = 1
                queue.push(next)
            }
        }
    }
    return seen.reduce((left, joined) => left + (joined ? 0 : 1), 0)
}

function piecesOf(tiles, {at, to, tileOf}) {
    const parent = Int32Array.from({length: tiles}, (_, t) => t)
    const root = (t) => {
        while (parent[t] !== t) {
            parent[t] = parent[parent[t]]
            t = parent[t]
        }
        return t
    }
    const coast = new Uint8Array(tiles)
    for (let v = 0; v < tileOf.length; v++) {
        const t = tileOf[v]
        if (t < 0) continue
        for (let e = at[v]; e < at[v + 1]; e++) {
            const u = tileOf[to[e]]
            if (u < 0) coast[t] = 1
            else parent[root(u)] = root(t)
        }
    }

    const pieceOf = new Int32Array(tiles)
    const named = new Map()
    for (let t = 0; t < tiles; t++) {
        const r = root(t)
        if (!named.has(r)) named.set(r, named.size)
        pieceOf[t] = named.get(r)
    }
    return {pieceOf, pieces: named.size, coast}
}

function snap(positions, landings, point, name) {
    const [x, y, z] = positionOf(point)
    let best = -1
    let along = -2
    for (const t of landings) {
        const d = positions[t * 3] * x + positions[t * 3 + 1] * y + positions[t * 3 + 2] * z
        if (d > along) {
            along = d
            best = t
        }
    }
    const km = Math.acos(Math.min(1, along)) * EARTH_KM
    if (best < 0 || km > SNAP_KM) {
        throw new Error(`${name}: [${point}] is ${Math.round(km)} km from the nearest coast, past ${SNAP_KM}`)
    }
    return best
}
