import {SEA} from "./ground.mjs"
import {lonLatOf} from "./lattice.mjs"

/**
 * One tile for each country no lattice vertex falls on. The lattice is 25 km apart, so the Vatican, Monaco, the
 * Maldives or Bahrain fall between its vertices and would have no ground at all. A seat names the one vertex that
 * becomes the country's, and never more than one: an archipelago is one tile, not a tile per island, because a
 * tile per island would spread it far past its own waters.
 *
 * Two kinds, so that a tile never sits on what the globe draws as open sea when the country is on a continent:
 *
 * - **On land** (`takes`): the country's neighbour gives up its vertex nearest `at`. `coast` says whether that vertex
 *   touches the sea, as the country does or does not: Monaco stays on the shore, San Marino stays inland.
 * - **An island** (no `takes`): the vertex nearest `at`, which is sea to Natural Earth because the island is smaller
 *   than a cell. `at` is the middle of the land in the cell that holds the most of the island the capital is on, so
 *   the tile sits over the island itself; `npm run earth` then paints the cell as land.
 *
 * The points are pinned, like the Natural Earth tag: moving one moves a tile.
 *
 * @type {Record<string, {at: [number, number], takes?: string, coast?: boolean}>}
 */
export const SEATS = {
    // Inside Rome, ringed by Italy as the city is.
    va: {at: [12.453, 41.903], takes: "it", coast: false},
    // Natural Earth's nearest Italian vertex is on the coast at Rimini; San Marino has no sea.
    sm: {at: [12.458, 43.942], takes: "it", coast: false},
    // On the Rhine, touching Switzerland and Austria.
    li: {at: [9.555, 47.146], takes: "ch", coast: false},
    // Between Nice and the Italian border, on the shore.
    mc: {at: [7.42, 43.74], takes: "fr", coast: true},
    // On Spain's shore: the nearest vertex is in the strait and touches Morocco too.
    gi: {at: [-5.35, 36.13], takes: "es", coast: true},
    mo: {at: [113.55, 22.16], takes: "cn", coast: true},

    ai: {at: [-63.058, 18.225]},
    aw: {at: [-69.99, 12.534]},
    bb: {at: [-59.507, 13.149]},
    bh: {at: [50.567, 26.135]},
    bm: {at: [-64.785, 32.286]},
    // South Keeling, where both villages are.
    cc: {at: [96.893, -12.188]},
    ck: {at: [-159.783, -21.222]}, // Rarotonga
    cx: {at: [105.662, -10.494]},
    gd: {at: [-61.656, 12.095]},
    gg: {at: [-2.557, 49.473]},
    // The cell of Guam's north holds the most of it; Hagåtña is a cell south.
    gu: {at: [144.872, 13.57]},
    io: {at: [72.431, -7.301]}, // Diego Garcia
    kn: {at: [-62.722, 17.316]}, // Saint Kitts, not Nevis
    mh: {at: [171.285, 7.085]}, // Majuro
    // Saipan, the capital's island; Tinian puts more land in one cell.
    mp: {at: [145.748, 15.2]},
    ms: {at: [-62.185, 16.72]},
    mv: {at: [73.509, 4.19]}, // Malé's atoll
    nf: {at: [167.957, -29.033]},
    nr: {at: [166.933, -0.521]},
    nu: {at: [-169.885, -19.049]},
    // Pitcairn Island, where everyone lives; Henderson is bigger and empty.
    pn: {at: [-130.104, -25.07]},
    pw: {at: [134.514, 7.377]}, // Koror and the south of Babeldaob
    sc: {at: [55.483, -4.704]}, // Mahé
    // Grand Turk, the capital's island; the Caicos have more land.
    tc: {at: [-71.14, 21.47]},
    // Fakaofo; Tokelau's capital rotates between its three atolls.
    tk: {at: [-171.196, -9.344]},
    to: {at: [-175.243, -21.172]}, // Tongatapu
    tv: {at: [179.206, -8.508]}, // Funafuti
    vi: {at: [-64.811, 18.338]}, // Saint Thomas and Saint John
}

/** Flags of a region inside a country's ground: they share it and have no seat. */
export const REGIONS = new Set(["gb-eng", "gb-sct", "gb-wls", "xb"])

/**
 * The oracle's answer for every lattice vertex: Natural Earth's, then each seat.
 * @param {{groundOf: (lon: number, lat: number) => string}} ground
 * @param {{count: number, positions: Float32Array, uvs: Float32Array}} vertices the whole lattice
 * @param {{at: Uint32Array, to: Uint32Array}} edges the lattice's, from `neighbours`
 * @returns {{grounds: string[], seated: {code: string, vertex: number, was: string}[]}}
 */
export function latticeGround(ground, {count, positions, uvs}, edges, seats = SEATS) {
    const grounds = new Array(count)
    for (let v = 0; v < count; v++) grounds[v] = ground.groundOf(...lonLatOf(uvs[v * 2], uvs[v * 2 + 1]))
    return {grounds, seated: seat(grounds, positions, edges, seats)}
}

/**
 * Gives each seat its vertex, in place.
 * @param {string[]} grounds one per lattice vertex
 * @param {Float32Array} positions
 * @param {{at: Uint32Array, to: Uint32Array}} edges
 * @param {typeof SEATS} seats
 * @returns {{code: string, vertex: number, was: string}[]} `was` is the ground the vertex had: SEA, or the neighbour's
 */
export function seat(grounds, positions, {at, to}, seats = SEATS) {
    const held = new Set(grounds)
    const touchesSea = (v) => {
        for (let e = at[v]; e < at[v + 1]; e++) if (grounds[to[e]] === SEA) return true
        return false
    }

    const placed = []
    const owner = new Map()
    for (const [code, {at: point, takes, coast}] of Object.entries(seats)) {
        if (held.has(code)) throw new Error(`${code} has ground of its own: a seat is for a country the lattice misses`)
        if (takes !== undefined && typeof coast !== "boolean") throw new Error(`${code} takes ${takes}'s ground: say whether it is on the coast`)

        const allowed = takes === undefined
            ? () => true
            : (v) => grounds[v] === takes && touchesSea(v) === coast
        const vertex = nearest(positions, point, allowed)
        if (vertex < 0) throw new Error(`${code}: no vertex of ${takes} ${coast ? "on" : "off"} the coast`)
        if (takes === undefined && grounds[vertex] !== SEA) {
            throw new Error(`${code}'s seat lands on ${grounds[vertex]}'s ground: name it in \`takes\`, or move the seat`)
        }
        if (owner.has(vertex)) throw new Error(`${code} and ${owner.get(vertex)} are seated on one vertex`)

        owner.set(vertex, code)
        placed.push({code, vertex, was: grounds[vertex]})
    }

    for (const {code, vertex} of placed) grounds[vertex] = code
    return placed
}

/** The unit vector of a point, as the lattice's positions are laid out. */
export function positionOf([lon, lat]) {
    const a = lon * Math.PI / 180, b = lat * Math.PI / 180
    return [-Math.cos(b) * Math.cos(a), Math.sin(b), Math.cos(b) * Math.sin(a)]
}

function nearest(positions, point, allowed) {
    const [x, y, z] = positionOf(point)
    let best = -1
    let bestDistance = Infinity
    for (let v = 0; v < positions.length / 3; v++) {
        const d = (positions[v * 3] - x) ** 2 + (positions[v * 3 + 1] - y) ** 2 + (positions[v * 3 + 2] - z) ** 2
        if (d < bestDistance && allowed(v)) {
            bestDistance = d
            best = v
        }
    }
    return best
}
