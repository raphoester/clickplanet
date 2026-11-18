// What a board's size scales (docs/boards.md, "Rules at the new density"): a number tuned on earth-1 reads the board
// instead. A count of tiles goes with its density, a length with its spacing, a zoom the other way. The backend and the
// game page apply the same rule, and on earth-1 every number comes out as it was, to the bit.

// earth-1's median spacing, in radians: what every such number was tuned on.
export const EARTH_1_SPACING = 0.004063093462255102

export type Scale = {
    // How much farther apart its tiles are than earth-1's: board.spacing / EARTH_1_SPACING.
    lengthRatio: number
    // How many more tiles it has on the same ground: (EARTH_1_SPACING / board.spacing)².
    densityRatio: number
    // A count of tiles on earth-1, on this board: never below one.
    tiles: (n: number) => number
    // A length on earth-1 (in radians, degrees or tile spacings of earth-1), on this board.
    length: (x: number) => number
    // A zoom on earth-1, on this board: a coarser board is seen from farther.
    zoom: (z: number) => number
}

export function scaleOf(board: {spacing: number}): Scale {
    if (!(board.spacing > 0 && Number.isFinite(board.spacing))) throw new Error(`a board's spacing is ${board.spacing}`)
    const lengthRatio = board.spacing / EARTH_1_SPACING
    const inverse = EARTH_1_SPACING / board.spacing
    const densityRatio = inverse * inverse
    return {
        lengthRatio,
        densityRatio,
        tiles: n => Math.max(1, Math.round(n * densityRatio)),
        length: x => x * lengthRatio,
        zoom: z => z / lengthRatio,
    }
}
