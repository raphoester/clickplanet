// A flag this many tiles from holding a whole landmass, and holding most of it, is close to fortifying it.
export const CLOSE_TILES = 25

export type Target = {
    flag: string
    missing: number
}

export function targetOf(size: number, leader: string | undefined, held: number, fortifiedBy: string | undefined): Target | undefined {
    if (leader === undefined || leader === fortifiedBy) return undefined
    const missing = size - held
    if (missing < 1 || missing > CLOSE_TILES || held * 2 <= size) return undefined
    return {flag: leader, missing}
}
