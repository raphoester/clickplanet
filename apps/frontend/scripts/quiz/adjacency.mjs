import {keyOf, lattice, neighbours} from "../map/lattice.mjs"
import {readBorders, readCoordinates} from "../map/blob.mjs"

/**
 * @param {string} [directory] where the two blobs are; /map by default
 * @returns {Map<string, Set<string>>} country code to the codes it touches
 */
export function adjacency(directory) {
    const tiles = readCoordinates(directory)
    const borders = readBorders(directory)

    if (borders.count !== tiles.count) {
        throw new Error(`${borders.name} covers ${borders.count} tiles, ${tiles.name} has ${tiles.count}`)
    }

    const countryOfTile = new Array(tiles.count)
    for (let t = 0; t < tiles.count; t++) countryOfTile[t] = borders.codes[borders.landmass[t]]

    const tileOfVertex = matchTilesToLattice(tiles)
    const {at, to} = neighbours()

    const touching = new Map()
    const link = (a, b) => {
        let own = touching.get(a)
        if (own === undefined) touching.set(a, own = new Set())
        own.add(b)
    }

    for (let v = 0; v < tileOfVertex.length; v++) {
        const tile = tileOfVertex[v]
        if (tile < 0) continue

        const mine = countryOfTile[tile]
        for (let e = at[v]; e < at[v + 1]; e++) {
            const other = tileOfVertex[to[e]]
            if (other < 0) continue

            const theirs = countryOfTile[other]
            if (theirs === mine) continue

            link(mine, theirs)
            link(theirs, mine)
        }
    }

    return touching
}

function matchTilesToLattice({count, positions, name}) {
    const {count: vertices, positions: latticePositions} = lattice()

    const vertexAt = new Map()
    for (let v = 0; v < vertices; v++) {
        vertexAt.set(keyOf(latticePositions[v * 3], latticePositions[v * 3 + 1], latticePositions[v * 3 + 2]), v)
    }

    const tileOfVertex = new Int32Array(vertices).fill(-1)
    for (let t = 0; t < count; t++) {
        const vertex = vertexAt.get(keyOf(positions[t * 3], positions[t * 3 + 1], positions[t * 3 + 2]))
        if (vertex === undefined) throw new Error(`${name}: tile ${t} is not a vertex of the lattice`)
        tileOfVertex[vertex] = t
    }

    return tileOfVertex
}
