export function tilePointSize(zoom: number, viewportHeight: number): number {
    return zoom * 1.5 * (viewportHeight / 1000)
}

// SCRATCH R&D. Tiles sit ~1.98px apart at zoom 1 on a 1000px-tall globe, so at
// 1.5 they never touch: the field is 76% covered at every zoom, which is what
// leaves the zoomed-out globe a dither instead of a surface. Circles on a hex
// lattice cover it fully at 1.155x the spacing — but only *far out*, where the
// tiles are meant to read as one painted skin. Zoomed in, a tile is a thing you
// aim at, and fattening it there just makes the discs collide, so the widening
// is undone by the time a tile is big enough to see.
const SPREAD = 2.3 / 1.5

export function displayPointSize(zoom: number, viewportHeight: number): number {
    const base = tilePointSize(zoom, viewportHeight)
    const grown = Math.min(1, Math.max(0, (base - 2.5) / 3.5))
    return base * (SPREAD + (1 - SPREAD) * grown)
}

// 1 = the flag is worth drawing, 0 = the tile is a flat patch of its colour.
export function flagDetail(displaySize: number): number {
    return Math.min(1, Math.max(0, (displaySize - 2.5) / 2))
}

export const MAX_PICK_WINDOW = 257

export function pickWindowSize(pointSize: number): number {
    const needed = Math.ceil(pointSize) + 1
    const odd = needed % 2 === 0 ? needed + 1 : needed
    return Math.min(Math.max(odd, 3), MAX_PICK_WINDOW)
}
