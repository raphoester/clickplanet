/**
 * The median arc between two touching tiles, in radians: a board's spacing.
 * @param {ArrayLike<number>} positions the tiles'
 * @param {{at: Uint32Array, to: Uint32Array, tileOf: Int32Array}} lattice from `tileLattice`
 * @returns {number}
 */
export function medianSpacingOf(positions, {at, to, tileOf}) {
    const arcs = []
    for (let v = 0; v < tileOf.length; v++) {
        const tile = tileOf[v]
        if (tile < 0) continue
        for (let e = at[v]; e < at[v + 1]; e++) {
            const other = tileOf[to[e]]
            if (other <= tile) continue
            const dx = positions[tile * 3] - positions[other * 3]
            const dy = positions[tile * 3 + 1] - positions[other * 3 + 1]
            const dz = positions[tile * 3 + 2] - positions[other * 3 + 2]
            arcs.push(2 * Math.asin(Math.hypot(dx, dy, dz) / 2))
        }
    }
    if (arcs.length === 0) throw new Error("no two tiles touch")
    return Float64Array.from(arcs).sort()[Math.floor(arcs.length / 2)]
}

/**
 * @param {Float32Array} positions the whole lattice, not just the land
 * @param {{at: Uint32Array, to: Uint32Array, count: number}} edges
 * @returns {number} radians
 */
export function spacingOf(positions, {at, to, count}) {
    let total = 0
    let taken = 0
    for (let v = 0; v < count; v++) {
        for (let e = at[v]; e < at[v + 1]; e++) {
            const other = to[e]
            if (other < v) continue
            const dx = positions[v * 3] - positions[other * 3]
            const dy = positions[v * 3 + 1] - positions[other * 3 + 1]
            const dz = positions[v * 3 + 2] - positions[other * 3 + 2]
            total += 2 * Math.asin(Math.hypot(dx, dy, dz) / 2)
            taken++
        }
    }
    return total / taken
}
