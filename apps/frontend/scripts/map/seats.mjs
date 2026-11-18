import {SEA} from "./ground.mjs"

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
 *   the tile sits over the island itself; `npm run terrain` then paints the cell as land.
 *
 * The points are pinned, like the Natural Earth tag: moving one moves a tile.
 *
 * On a lattice coarser than the gap between two islands, both can be nearest one vertex. `beside` names the seat
 * earlier in the table that keeps it: this one takes the nearest vertex that seat did not (Guernsey beside Jersey on
 * `siege-1`). Two seats on one vertex that no `beside` names refuse the generation.
 *
 * @type {Record<string, {at: [number, number], takes?: string, coast?: boolean, beside?: string}>}
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

const earth1s = (codes) => Object.fromEntries(codes.map((code) => [code, SEATS[code]]))

/**
 * Each board's seats: a lattice of another spacing misses other countries. `earth-2`, at 12.5 km, gives 20 of earth-1's
 * 34 their own ground, and misses the British Virgin Islands, whose vertices fell on earth-1's lattice. `siege-1`, at
 * 79 km, misses 50 countries: earth-1's seats, and every country smaller than its cell. A seat earth-1 has keeps its
 * point; a new one is at its capital, which on a cell 79 km wide is the middle of the land the cell holds.
 * @type {Record<string, typeof SEATS>}
 */
export const SEATS_BY_BOARD = {
    "earth-1": SEATS,
    "earth-2": {
        ...earth1s(["va", "sm", "gi", "mo", "cc", "ck", "mv", "ms", "nr", "nf", "pn", "tk", "tv"]),
        // Natural Earth draws the Indian Ocean Territories inside Australia, and a vertex of earth-2 falls on Christmas
        // Island: it is Australia's ground, given to cx.
        cx: {...SEATS.cx, takes: "au", coast: true},
        vg: {at: [-64.623, 18.428]}, // Road Town, Tortola
    },
    "siege-1": {
        ...earth1s([
            "va", "sm", "li", "mc", "gi", "mo",
            "ai", "aw", "bb", "bh", "bm", "cc", "ck", "cx", "gd", "gu", "kn", "mh", "mp", "ms", "mv", "nf", "nr",
            "nu", "pn", "pw", "sc", "tc", "tk", "to", "tv", "vi",
        ]),
        lu: {at: [6.13, 49.612], takes: "fr", coast: false},
        // Macao takes the coast at Zhuhai; Hong Kong's nearest vertex is in the water off the New Territories, and its
        // cell holds the city.
        hk: {at: [114.183, 22.307]},
        as: {at: [-170.707, -14.277]}, // Pago Pago
        ag: {at: [-61.85, 17.118]}, // Saint John's
        ky: {at: [-81.33, 19.28]}, // George Town
        cw: {at: [-69.02, 12.2]}, // Willemstad
        pf: {at: [-149.567, -17.533]}, // Papeete
        im: {at: [-4.48, 54.15]}, // Douglas
        je: {at: [-2.107, 49.187]}, // Saint Helier
        // Jersey is 20 km from the vertex both islands are nearest; Guernsey takes the next, 48 km north-west.
        gg: {...SEATS.gg, beside: "je"},
        ki: {at: [173.018, 1.338]}, // Tarawa
        mt: {at: [14.515, 35.9]}, // Valletta
        fm: {at: [158.15, 6.917]}, // Palikir
        lc: {at: [-61, 14.002]}, // Castries
        vc: {at: [-61.212, 13.148]}, // Kingstown
        st: {at: [6.733, 0.333]}, // São Tomé
        sg: {at: [103.854, 1.295]},
        sx: {at: [-63.047, 18.026]}, // Philipsburg
        // The US Virgin Islands keep the vertex between them; Tortola's next is 47 km north of Road Town.
        vg: {at: [-64.623, 18.428], beside: "vi"}, // Road Town, Tortola
        wf: {at: [-176.174, -13.282]}, // Mata-Utu, Wallis
    },
}

/** @param {string} board @returns {typeof SEATS} */
export function seatsOf(board) {
    const seats = SEATS_BY_BOARD[board]
    if (seats === undefined) throw new Error(`no seats for board ${board}: name them in SEATS_BY_BOARD`)
    return seats
}

/** Flags of a region inside a country's ground: they share it and have no seat. */
export const REGIONS = new Set(["gb-eng", "gb-sct", "gb-wls", "xb"])

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
    const vertexOf = new Map()
    for (const [code, {at: point, takes, coast, beside}] of Object.entries(seats)) {
        if (held.has(code)) throw new Error(`${code} has ground of its own: a seat is for a country the lattice misses`)
        if (takes !== undefined && typeof coast !== "boolean") throw new Error(`${code} takes ${takes}'s ground: say whether it is on the coast`)
        if (beside !== undefined && !vertexOf.has(beside)) throw new Error(`${code} is beside ${beside}, which no seat before it is`)

        const kept = beside === undefined ? -1 : vertexOf.get(beside)
        const allowed = takes === undefined
            ? (v) => v !== kept
            : (v) => v !== kept && grounds[v] === takes && touchesSea(v) === coast
        const vertex = nearest(positions, point, allowed)
        if (vertex < 0) throw new Error(`${code}: no vertex of ${takes} ${coast ? "on" : "off"} the coast`)
        if (takes === undefined && grounds[vertex] !== SEA) {
            throw new Error(`${code}'s seat lands on ${grounds[vertex]}'s ground: name it in \`takes\`, or move the seat`)
        }
        if (owner.has(vertex)) throw new Error(`${code} and ${owner.get(vertex)} are seated on one vertex`)

        owner.set(vertex, code)
        vertexOf.set(code, vertex)
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
