export function tilePointSize(zoom: number, viewportHeight: number): number {
    return zoom * 1.5 * (viewportHeight / 1000)
}

// Tiles sit ~1.98px apart at zoom 1 on a 1000px-tall globe, so at 1.5 they
// never touch: the field is 76% covered at every zoom, which is what
// leaves the zoomed-out globe a dither instead of a surface. Circles on a hex
// lattice cover it fully at 1.155x the spacing, and 2.3/1.5 is that ratio.
const SPREAD = 2.3 / 1.5

// The handover from the flag painted across a landmass to the tiles themselves:
// 0 while the painted flag owns the frame, 1 once the tiles do. Measured in the
// tile's own point size rather than in zoom, because what it is really about is
// how big a tile is on screen, and that depends on the viewport too.
//
// One schedule drives all three parts of the handover — the flag fading out,
// the tiles fading in, and the widening being undone — because they only work
// together. The painted flag reaches the ground only through the discs, so
// while it is showing they have to cover the ground; and a tile you are about
// to aim at must not be fattened, so the widening has to be gone by then.
// Running them on separate schedules left a band where the discs had already
// shrunk back to 76% cover while the flag was still being painted through them,
// and the flag quietly lost a third of its ink there.
const COARSE_FROM = 5
const COARSE_UNTIL = 8

export function coarseHandover(tileSize: number): number {
    return Math.min(1, Math.max(0, (tileSize - COARSE_FROM) / (COARSE_UNTIL - COARSE_FROM)))
}

export function displayPointSize(zoom: number, viewportHeight: number): number {
    const base = tilePointSize(zoom, viewportHeight)
    return base * (SPREAD + (1 - SPREAD) * coarseHandover(base))
}

// 1 = the landmass wears its holder's flag, 0 = the tiles speak for themselves.
export function flagPaint(zoom: number, viewportHeight: number): number {
    return 1 - coarseHandover(tilePointSize(zoom, viewportHeight))
}

export const MAX_PICK_WINDOW = 257

export function pickWindowSize(pointSize: number): number {
    const needed = Math.ceil(pointSize) + 1
    const odd = needed % 2 === 0 ? needed + 1 : needed
    return Math.min(Math.max(odd, 3), MAX_PICK_WINDOW)
}
