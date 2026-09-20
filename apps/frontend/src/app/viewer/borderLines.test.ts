import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {borderSegments, createBorderLines, decodeBorderLines, OVER, UNDER, WIDTH} from "./borderLines.ts"
import {innerSphere} from "./sphere.ts"
import {displayPointSize, flagPaint, tileSpacing} from "./pointSize.ts"
import {MAX_ZOOM, MIN_ZOOM} from "./zoom.ts"

/** A blob as scripts/generateBorderLines.mjs writes one. */
function blobOf(runs: number[][]): ArrayBuffer {
    const corners = runs.flat()
    const bytes = new ArrayBuffer(16 + runs.length * 4 + corners.length * 2)
    const head = new DataView(bytes)
    for (const [at, code] of [..."CPBL"].entries()) head.setUint8(at, code.charCodeAt(0))
    head.setUint32(4, 1, true)
    head.setUint32(8, runs.length, true)
    head.setUint32(12, corners.length / 3, true)
    new Uint32Array(bytes, 16, runs.length).set(runs.map((run) => run.length / 3))
    new Int16Array(bytes, 16 + runs.length * 4).set(corners)
    return bytes
}

describe("decodeBorderLines", () => {
    it("reads back the runs and the corners", () => {
        const data = decodeBorderLines(blobOf([[1, 2, 3, 4, 5, 6], [7, 8, 9, 10, 11, 12, 13, 14, 15]]))
        expect([...data.runs]).toEqual([2, 3])
        expect([...data.corners]).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15])
    })

    it("refuses a blob that is not one", () => {
        const bytes = blobOf([[1, 2, 3, 4, 5, 6]])
        new DataView(bytes).setUint8(0, "X".charCodeAt(0))
        expect(() => decodeBorderLines(bytes)).toThrow(/magic/)
    })

    it("refuses a version it does not know", () => {
        const bytes = blobOf([[1, 2, 3, 4, 5, 6]])
        new DataView(bytes).setUint32(4, 2, true)
        expect(() => decodeBorderLines(bytes)).toThrow(/version 2/)
    })
})

describe("borderSegments", () => {
    it("draws one edge between each pair of corners in a run", () => {
        const {from, to} = borderSegments(decodeBorderLines(blobOf([[1, 2, 3, 4, 5, 6, 7, 8, 9]])))
        expect([...from]).toEqual([1, 2, 3, 4, 5, 6])
        expect([...to]).toEqual([4, 5, 6, 7, 8, 9])
    })

    it("never joins the end of one run to the start of the next", () => {
        const {from, to} = borderSegments(decodeBorderLines(blobOf([
            [1, 1, 1, 2, 2, 2],
            [9, 9, 9, 8, 8, 8],
        ])))
        expect([...from]).toEqual([1, 1, 1, 9, 9, 9])
        expect([...to]).toEqual([2, 2, 2, 8, 8, 8])
    })
})

describe("the two passes", () => {
    // The tiles are drawn on the sphere of radius 1 and the earth's own texture
    // just under it. The under pass has to fall between the two: below it, the
    // earth swallows the outline whole; at or above the tiles, nothing they
    // cover is ever hidden and the pass is pointless.
    it("sit either side of the tiles, and above the earth", () => {
        const earth = innerSphere().parameters.radius
        expect(UNDER).toBeGreaterThan(earth)
        expect(UNDER).toBeLessThan(1)
        expect(OVER).toBeGreaterThan(1)
    })
})

describe("the handover", () => {
    const outline = () => {
        const lines = createBorderLines(decodeBorderLines(blobOf([[1, 2, 3, 4, 5, 6, 7, 8, 9]])))
        const [over, under] = lines.object.children as THREE.Mesh[]
        const inkOf = (pass: THREE.Mesh) =>
            pass.visible ? (pass.material as THREE.ShaderMaterial).uniforms.ink.value as number : 0
        return {lines, over, under, inkOf}
    }

    it("hands the outline from over the tiles to under them, and never drops it", () => {
        const {lines, over, under, inkOf} = outline()

        for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
            lines.update(zoom, 1600, 900)

            // Zoomed out the tiles are widened until they cover the ground, so
            // an outline under them would be hidden; zoomed in they part and it
            // is the over pass that would cut across them. In between both are
            // drawn, and the line is never fainter than either end of the
            // handover — a border that faded out halfway through a zoom would
            // be the one thing this is for.
            const painted = flagPaint(zoom, 900)
            expect(inkOf(over), `over at zoom ${zoom}`).toBe(painted)
            expect(inkOf(under) + inkOf(over), `ink at zoom ${zoom}`).toBeGreaterThanOrEqual(1)
        }

        lines.dispose()
    })

    it("fits in the gap the tiles leave, wherever it is the only line drawn", () => {
        // What the whole thing rests on: the outline runs along the cell edges
        // of the tile lattice, so once the discs are back to their own size it
        // passes down the middle of the gap between two of them rather than
        // over either. A line wider than that gap would be back to cutting the
        // tiles it is drawn between.
        for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
            if (flagPaint(zoom, 900) > 0) continue
            const gap = tileSpacing(zoom, 900) - displayPointSize(zoom, 900)
            expect(WIDTH, `zoom ${zoom}`).toBeLessThan(gap)
        }
    })

    it("draws nothing it cannot see", () => {
        const {lines, over, under} = outline()

        lines.update(MIN_ZOOM, 1600, 900)
        expect([over.visible, under.visible]).toEqual([true, false])

        lines.update(MAX_ZOOM, 1600, 900)
        expect([over.visible, under.visible]).toEqual([false, true])

        lines.dispose()
    })

    it("measures the line in the drawing buffer's own pixels", () => {
        const {lines, over, under} = outline()
        lines.update(1, 1600, 900)

        for (const pass of [over, under]) {
            const halfViewport = (pass.material as THREE.ShaderMaterial).uniforms.halfViewport.value as THREE.Vector2
            expect([halfViewport.x, halfViewport.y]).toEqual([800, 450])
        }

        lines.dispose()
    })
})
