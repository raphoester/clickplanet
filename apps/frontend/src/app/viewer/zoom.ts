/**
 * How far the orthographic camera may pull back and push in.
 *
 * The camera's half-height is `1 / zoom` against a globe of radius 1, so the
 * zoom reads directly as the share of the viewport's height the globe takes up:
 * 1 fills it edge to edge, 0.5 fills half of it.
 */

/**
 * The globe at half the viewport's height, which is as far out as the view
 * goes. At 1 the sphere is pressed against the top and bottom of the screen —
 * its atmosphere is clipped off there and there is no sky to either side of it,
 * so the planet never reads as an object you are looking at from somewhere.
 */
export const MIN_ZOOM = 0.5

/** Close enough that a single tile is a disc you can aim at. */
export const MAX_ZOOM = 50

/**
 * The zoom the view opens at, and the line between looking at the planet and
 * looking into it: at or below this the whole globe is in frame and the idle
 * spin keeps turning, above it the view is held where it was put.
 */
export const RESTING_ZOOM = 1
