import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {
    choreograph,
    createEnclosureEffects,
    FADE_FROM,
    FILL_FROM,
    FILL_SECONDS,
    LIFETIME_SECONDS,
    markLook,
    OUTLINE_SECONDS,
    WAVE_SECONDS,
    waveLook,
} from "./enclosureEffect.ts"
import type {Enclosure} from "../../backends/backend.ts"

// A hexagon of six tiles around tile 1, on the ground facing +Z. Tile 2 is due
// east of the middle, and the rest follow it round.
const STEP = 0.01
const around = Array.from({length: 6}, (_, i) => {
    const angle = (i * Math.PI) / 3
    return new THREE.Vector3(Math.cos(angle) * STEP, Math.sin(angle) * STEP, 1).normalize()
})
const positions = new Float32Array([0, 0, 1, ...around.flatMap(p => [p.x, p.y, p.z])])

const ring: Enclosure = {
    countryId: "fr",
    closingTile: 2,
    wall: [2, 3, 4, 5, 6, 7],
    filled: [1],
}

const startOf = (enclosure: Enclosure, tile: number) =>
    choreograph(enclosure, positions).marks.find(mark => mark.tile === tile)!.start

describe("choreograph", () => {
    it("finds the middle of the shape and how far it reaches", () => {
        const {centre, radius} = choreograph(ring, positions)

        expect(centre.distanceTo(new THREE.Vector3(0, 0, 1))).toBeLessThan(1e-6)
        expect(radius).toBeCloseTo(STEP, 3)
    })

    it("lights the far side of the outline first and the closing tile last", () => {
        // Tile 5 is straight across from tile 2.
        expect(startOf(ring, 5)).toBeCloseTo(0, 6)
        expect(startOf(ring, 2)).toBe(OUTLINE_SECONDS)

        for (const tile of [3, 4, 6, 7]) {
            expect(startOf(ring, tile)).toBeGreaterThan(0)
            expect(startOf(ring, tile)).toBeLessThan(OUTLINE_SECONDS)
        }
    })

    it("runs the light round both sides at once, so they meet at the closing tile", () => {
        expect(startOf(ring, 3)).toBeCloseTo(startOf(ring, 7), 6)
        expect(startOf(ring, 4)).toBeCloseTo(startOf(ring, 6), 6)
        expect(startOf(ring, 3)).toBeGreaterThan(startOf(ring, 4))
    })

    it("pours the inside in in the order the server found it, once the outline has closed", () => {
        const wide: Enclosure = {...ring, wall: [2, 3, 4], filled: [5, 6, 7, 1]}
        const starts = wide.filled.map(tile => startOf(wide, tile))

        expect(starts[0]).toBe(FILL_FROM)
        expect(starts.at(-1)).toBeCloseTo(FILL_FROM + FILL_SECONDS, 6)
        expect([...starts].sort((a, b) => a - b)).toEqual(starts)
        expect(FILL_FROM).toBeGreaterThanOrEqual(OUTLINE_SECONDS)
    })

    it("marks every tile of the shape exactly once", () => {
        const {marks} = choreograph(ring, positions)

        expect(marks.map(mark => mark.tile).sort()).toEqual([1, 2, 3, 4, 5, 6, 7])
        expect(marks.filter(mark => mark.role === "closing")).toHaveLength(1)
    })
})

describe("markLook", () => {
    const wall = {tile: 3, start: 0.2, role: "wall"} as const
    const closing = {tile: 2, start: OUTLINE_SECONDS, role: "closing"} as const

    it("draws nothing before a tile's turn", () => {
        expect(markLook(wall, 0.1).glow).toBe(0)
        expect(markLook(wall, 0.1).scale).toBe(0)
    })

    it("flashes when its turn comes, then settles into a glow", () => {
        const lit = markLook(wall, wall.start)
        const settled = markLook(wall, wall.start + 1)

        expect(lit.white).toBeCloseTo(1, 6)
        expect(lit.scale).toBeGreaterThan(settled.scale)
        expect(settled.glow).toBeGreaterThan(0)
    })

    it("makes the tile that closed the shape the biggest flash of all", () => {
        expect(markLook(closing, closing.start).scale).toBeGreaterThan(markLook(wall, wall.start).scale)
    })

    it("holds, then fades to nothing by the end", () => {
        expect(markLook(wall, FADE_FROM).glow).toBeGreaterThan(0)
        expect(markLook(wall, LIFETIME_SECONDS - 0.01).glow).toBeLessThan(0.01)
        expect(markLook(wall, LIFETIME_SECONDS).glow).toBe(0)
    })

    it("still lights up for less motion, without flashing or swelling", () => {
        const filled = {tile: 1, start: FILL_FROM, role: "filled"} as const

        const early = markLook(filled, FILL_FROM, true)
        const late = markLook(filled, FILL_FROM + 0.8, true)

        expect(early.glow).toBeGreaterThan(0)
        expect(early.white).toBe(late.white)
        expect(early.scale).toBe(late.scale)
    })
})

describe("waveLook", () => {
    it("runs only for its own stretch of the effect", () => {
        expect(waveLook(0.5, 0.49)).toBeUndefined()
        expect(waveLook(0.5, 0.5)).toBeDefined()
        expect(waveLook(0.5, 0.5 + WAVE_SECONDS)).toBeUndefined()
    })

    it("runs outwards and fades as it goes", () => {
        const early = waveLook(0, 0.1)!
        const late = waveLook(0, WAVE_SECONDS * 0.9)!

        expect(late.radius).toBeGreaterThan(early.radius)
        expect(late.opacity).toBeLessThan(early.opacity)
        expect(late.radius).toBeLessThanOrEqual(1)
    })
})

describe("createEnclosureEffects", () => {
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1)

    it("starts a shape on the next frame and takes it off once it is over", () => {
        const effects = createEnclosureEffects(positions)

        effects.play(ring)
        expect(effects.object.children.length).toBeGreaterThan(0)

        // The first frame is the start, however late the clock already is.
        effects.update(1000, camera, 800)
        effects.update(1000 + LIFETIME_SECONDS / 2, camera, 800)
        expect(effects.object.children.length).toBeGreaterThan(0)

        effects.update(1000 + LIFETIME_SECONDS, camera, 800)
        expect(effects.object.children).toHaveLength(0)

        effects.dispose()
    })

    it("keeps a burst of shapes to a bounded number on screen", () => {
        const effects = createEnclosureEffects(positions)

        effects.play(ring)
        const perShape = effects.object.children.length
        for (let i = 0; i < 50; i++) effects.play(ring)

        expect(effects.object.children.length).toBeLessThan(perShape * 50)
        effects.dispose()
        expect(effects.object.children).toHaveLength(0)
    })
})
