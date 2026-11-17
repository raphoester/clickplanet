import {boardDir, readBoard, readBorders, tileOfPoint} from "../map/board.mjs"
import {neighbours} from "../map/lattice.mjs"

/**
 * @param {string} [directory] the board's; earth-1's by default
 * @returns {Map<string, Set<string>>} country code to the codes it touches
 */
export function adjacency(directory = boardDir()) {
    const board = readBoard(directory)
    const borders = readBorders(board)

    if (borders.count !== board.count) {
        throw new Error(`${borders.name} covers ${borders.count} tiles, ${board.file} has ${board.count}`)
    }

    const countryOfTile = new Array(board.count)
    for (let t = 0; t < board.count; t++) countryOfTile[t] = borders.codes[borders.landmass[t]]

    const {count, at, to} = neighbours(board.manifest.lattice.cols)
    const tileOfVertex = tileOfPoint(board, count)

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
