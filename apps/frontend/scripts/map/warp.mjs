import {CORNERS, latticeSize, latticeTriangles, positionWriter, walkLattice} from "../../src/domain/lattice.ts"

/**
 * The rotation that puts the twelve five-neighbour points at sea, each at least 184 km from an `earth-1` tile: found by
 * a random search over rotations, then refined, maximising the smallest distance to land (docs/boards.md). A corner `c`
 * goes to `R·c`. Readers never see it: a board's manifest carries the corners it gives (`rotatedCorners`).
 */
export const ROTATION = [
    [0.09926508736568841, -0.49847683751896155, -0.8612010711136958],
    [0.5914752627857613, 0.725540318816103, -0.35177870783312387],
    [0.8001896374877626, -0.47445978568190805, 0.36685754159875256],
]

/**
 * Three's twelve unnormalised corners turned by `rotation`, as the 36 float64s a manifest's `corners` holds: each
 * coordinate `(R[r][0]·x + R[r][1]·y) + R[r][2]·z`.
 * @param {number[][]} rotation
 * @returns {number[]}
 */
export function rotatedCorners(rotation = ROTATION) {
    const corners = []
    for (let corner = 0; corner < 12; corner++) {
        const x = CORNERS[corner * 3], y = CORNERS[corner * 3 + 1], z = CORNERS[corner * 3 + 2]
        for (const row of rotation) corners.push(row[0] * x + row[1] * y + row[2] * z)
    }
    return corners
}

/**
 * The ground each lattice point's cell covers, in steradians, in the lattice's order: a third of each of its triangles.
 * @param {import("../../src/domain/lattice.ts").Lattice} lattice
 * @param {Uint32Array} [triangles] the lattice's, from `latticeTriangles`, to share between calls
 * @returns {Float64Array}
 */
export function cellAreas(lattice, triangles = latticeTriangles(lattice)) {
    const count = latticeSize(lattice.cols)
    const positions = new Float32Array(count * 3)
    const place = positionWriter(lattice)
    walkLattice(lattice, (point, face, i, j) => place(face, i, j, positions, point * 3))

    const areas = new Float64Array(count)
    for (let t = 0; t < triangles.length; t += 3) {
        const a = triangles[t] * 3, b = triangles[t + 1] * 3, c = triangles[t + 2] * 3
        const ax = positions[a], ay = positions[a + 1], az = positions[a + 2]
        const bx = positions[b], by = positions[b + 1], bz = positions[b + 2]
        const cx = positions[c], cy = positions[c + 1], cz = positions[c + 2]
        // Van Oosterom and Strackee: tan(E / 2) = |a·(b×c)| / (1 + a·b + b·c + c·a).
        const triple = ax * (by * cz - bz * cy) - ay * (bx * cz - bz * cx) + az * (bx * cy - by * cx)
        const below = 1 + (ax * bx + ay * by + az * bz) + (bx * cx + by * cy + bz * cz) + (cx * ax + cy * ay + cz * az)
        const third = 2 * Math.atan2(Math.abs(triple), below) / 3
        areas[triangles[t]] += third
        areas[triangles[t + 1]] += third
        areas[triangles[t + 2]] += third
    }
    return areas
}

/**
 * The spread of the cells' areas: the largest over the smallest, the twelve five-neighbour corners left out, since
 * their cells are five triangles, not six, and they are at sea.
 * @param {import("../../src/domain/lattice.ts").Lattice} lattice
 * @param {Uint32Array} [triangles]
 * @returns {number}
 */
export function areaSpread(lattice, triangles) {
    const areas = cellAreas(lattice, triangles)
    const corner = new Uint8Array(areas.length)
    walkLattice(lattice, (point, _face, _i, _j, id) => {
        if (id < 12) corner[point] = 1
    })
    let least = Infinity, most = 0
    for (let point = 0; point < areas.length; point++) {
        if (corner[point]) continue
        least = Math.min(least, areas[point])
        most = Math.max(most, areas[point])
    }
    return most / least
}

// Where the fit starts: the coefficients fitted at `cols` 24 when the warp was designed.
export const FIRST_FIT = {a: -0.05520744, b: -0.16849639, c: 0.30479447}

const DIGITS = 8

/**
 * The warp whose cells' areas spread least at `cols`: Nelder-Mead over (a, b, c), from `start`, rounded to 8 decimals,
 * which the manifest keeps. Deterministic: the same `cols` and `start` give the same coefficients.
 * @param {number} cols
 * @param {{a: number, b: number, c: number}} [start]
 * @param {{steps?: number, say?: (line: string) => void}} [options]
 * @returns {{warp: {a: number, b: number, c: number}, spread: number, evaluations: number}}
 */
export function fitWarp(cols, start = FIRST_FIT, {steps = 160, say = () => {}} = {}) {
    const lattice = {cols, warp: null, corners: null, order: "canonical"}
    const triangles = latticeTriangles(lattice)
    let evaluations = 0
    const spreadAt = ([a, b, c]) => {
        evaluations++
        return areaSpread({...lattice, warp: {a, b, c}}, triangles)
    }

    const first = [start.a, start.b, start.c]
    const simplex = [first, ...[0, 1, 2].map((k) => first.map((value, at) => at === k ? value + 0.02 : value))]
        .map((point) => ({point, value: spreadAt(point)}))
    const towards = (from, to, by) => from.map((value, k) => value + (to[k] - value) * by)
    for (let step = 0; step < steps; step++) {
        simplex.sort((one, other) => one.value - other.value)
        const best = simplex[0], worst = simplex[3]
        const centre = [0, 1, 2].map((k) => (simplex[0].point[k] + simplex[1].point[k] + simplex[2].point[k]) / 3)
        const reflected = towards(worst.point, centre, 2)
        const reflectedValue = spreadAt(reflected)
        if (reflectedValue < best.value) {
            const expanded = towards(worst.point, centre, 3)
            const expandedValue = spreadAt(expanded)
            simplex[3] = expandedValue < reflectedValue ? {point: expanded, value: expandedValue} : {point: reflected, value: reflectedValue}
        } else if (reflectedValue < simplex[2].value) {
            simplex[3] = {point: reflected, value: reflectedValue}
        } else {
            const contracted = towards(worst.point, centre, 0.5)
            const contractedValue = spreadAt(contracted)
            if (contractedValue < worst.value) {
                simplex[3] = {point: contracted, value: contractedValue}
            } else {
                for (let k = 1; k < 4; k++) {
                    const point = towards(best.point, simplex[k].point, 0.5)
                    simplex[k] = {point, value: spreadAt(point)}
                }
            }
        }
        if (step % 20 === 0) say(`step ${step}: spread ${simplex[0].value.toFixed(6)}`)
    }
    simplex.sort((one, other) => one.value - other.value)
    const [a, b, c] = simplex[0].point.map((value) => Number(value.toFixed(DIGITS)))
    return {warp: {a, b, c}, spread: spreadAt([a, b, c]), evaluations}
}
