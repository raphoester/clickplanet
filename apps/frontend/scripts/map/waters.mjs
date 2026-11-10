/**
 * Each body of water the lattice draws, and the tiles on its coast. Water is every lattice vertex that is no tile, and
 * two water vertices that touch are one body, so a strait narrower than a cell cuts a sea in two: the Bosporus does.
 * A tile is on the coast of each body one of its vertex's neighbours is in, so a tile between two is on both.
 * @param {{at: Uint32Array, to: Uint32Array}} edges the lattice's, from `neighbours`
 * @param {Int32Array} tileOf each lattice vertex's tile index, or -1 for water
 * @returns {number[][]} each body's coast tiles as wire ids (index + 1), ascending; the longest coast first
 */
export function watersOf({at, to}, tileOf) {
    const vertices = tileOf.length
    const body = new Int32Array(vertices).fill(-1)
    const queue = new Int32Array(vertices)
    let bodies = 0
    for (let v = 0; v < vertices; v++) {
        if (tileOf[v] >= 0 || body[v] >= 0) continue
        let head = 0
        let tail = 0
        queue[tail++] = v
        body[v] = bodies
        while (head < tail) {
            const x = queue[head++]
            for (let e = at[x]; e < at[x + 1]; e++) {
                const y = to[e]
                if (tileOf[y] < 0 && body[y] < 0) {
                    body[y] = bodies
                    queue[tail++] = y
                }
            }
        }
        bodies++
    }

    const coasts = Array.from({length: bodies}, () => [])
    for (let v = 0; v < vertices; v++) {
        if (tileOf[v] < 0) continue
        const touched = new Set()
        for (let e = at[v]; e < at[v + 1]; e++) if (body[to[e]] >= 0) touched.add(body[to[e]])
        for (const b of touched) coasts[b].push(tileOf[v] + 1)
    }

    return coasts
        .map((coast, first) => ({coast: coast.sort((a, b) => a - b), first}))
        .filter(({coast}) => coast.length > 0)
        .sort((a, b) => b.coast.length - a.coast.length || a.first - b.first)
        .map(({coast}) => coast)
}
