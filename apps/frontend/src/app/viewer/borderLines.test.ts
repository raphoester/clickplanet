import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {
    COARSE_SAMPLES,
    createBorderLines,
    decodeBorderLines,
    outlineSegments,
    OVER,
    SAMPLES,
    UNDER,
    WIDTH,
} from "./borderLines.ts"
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

describe("outlineSegments", () => {
    const at = (points: Int16Array, piece: number) => [...points.slice(piece * 3, piece * 3 + 3)]

    it("starts and ends a run exactly on its own corners", () => {
        // A run ends where three countries meet, and the two other runs end
        // there too. That corner is the one the curve may not round off, or the
        // three of them stop short of each other by a fraction of a tile.
        const {from, to} = outlineSegments(decodeBorderLines(blobOf([[1000, 0, 0, 1000, 500, 0, 1000, 900, 0]])))
        expect(at(from, 0)).toEqual([1000, 0, 0])
        expect(at(to, to.length / 3 - 1)).toEqual([1000, 900, 0])
    })

    it("hands each piece on to the next with no seam between them", () => {
        const {from, to} = outlineSegments(decodeBorderLines(blobOf([[1000, 0, 0, 1000, 500, 0, 1000, 900, 300]])))
        for (let piece = 1; piece < from.length / 3; piece++) {
            expect(at(from, piece), `piece ${piece}`).toEqual(at(to, piece - 1))
        }
    })

    it("closes a run that came back to where it started", () => {
        const loop = [0, 0, 30000, 0, 30000, 0, 30000, 0, 0, 0, 0, 30000]
        const {from, to} = outlineSegments(decodeBorderLines(blobOf([loop])))
        expect(at(to, to.length / 3 - 1)).toEqual(at(from, 0))
    })

    it("never joins the end of one run to the start of the next", () => {
        const samples = 4
        const one = [0, 0, 30000, 0, 10000, 30000]
        const other = [30000, 0, 0, 30000, 10000, 0]
        const {from, to} = outlineSegments(decodeBorderLines(blobOf([one, other])), samples)

        // Two corners each, so the first run owns the first `samples` pieces
        // twice over and stops dead on its own last corner.
        expect(at(to, 2 * samples - 1)).toEqual([0, 10000, 30000])
        expect(at(from, 2 * samples)).toEqual([30000, 0, 0])
    })

    // The cell around a tile, at the lattice's own proportions: a regular
    // hexagon whose edges are half a tile spacing from the tile in the middle
    // of it, which is the narrowest the corridor between two tiles ever gets.
    const SPACING = 20000
    const cell = () => [0, 1, 2, 3, 4, 5, 0].flatMap((which) => {
        const angle = which * Math.PI / 3
        const reach = SPACING / Math.sqrt(3)
        return [Math.round(reach * Math.cos(angle)), Math.round(reach * Math.sin(angle)), 0]
    })

    /** How near the drawn outline comes to the tile it is drawn around. */
    const nearest = (samples: number) => {
        const {from, to} = outlineSegments(decodeBorderLines(blobOf([cell()])), samples)
        let closest = Infinity
        for (let piece = 0; piece < from.length / 3; piece++) {
            const [ax, ay] = at(from, piece)
            const [bx, by] = at(to, piece)
            const run = (bx - ax) ** 2 + (by - ay) ** 2
            const along = run === 0 ? 0 : Math.min(1, Math.max(0, -(ax * (bx - ax) + ay * (by - ay)) / run))
            closest = Math.min(closest, Math.hypot(ax + along * (bx - ax), ay + along * (by - ay)))
        }
        return closest
    }

    // What the smoothing may not cost, and the whole thing rests on it: at every
    // zoom where this outline is the only one drawn, the near edge of the line
    // has to clear the near edge of the disc it runs past. The curve is allowed
    // to round the corners off — it is not allowed to touch a tile.
    it("keeps the line clear of the tiles it runs between, at every zoom it is drawn alone", () => {
        const share = nearest(SAMPLES) / SPACING

        for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
            if (flagPaint(zoom, 900) > 0) continue
            const clearance = share * tileSpacing(zoom, 900) - WIDTH / 2
            expect(clearance, `zoom ${zoom}`).toBeGreaterThan(displayPointSize(zoom, 900) / 2)
        }
    })

    // The two passes are drawn over each other through the whole handover, so
    // the coarser one has to fall on the finer one rather than beside it.
    it("draws the coarse pass close enough to the fine one to cross-fade with it", () => {
        expect(Math.abs(nearest(COARSE_SAMPLES) - nearest(SAMPLES))).toBeLessThan(SPACING * 0.02)
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


    it("draws nothing it cannot see", () => {
        const {lines, over, under} = outline()

        lines.update(MIN_ZOOM, 1600, 900)
        expect([over.visible, under.visible]).toEqual([true, false])

        lines.update(MAX_ZOOM, 1600, 900)
        expect([over.visible, under.visible]).toEqual([false, true])

        lines.dispose()
    })

    // The coarse pass is the one on screen whenever the whole globe is, so it is
    // the one worth not drawing four times over.
    it("gives the coarse outline to the pass that is drawn zoomed out", () => {
        const {lines, over, under} = outline()
        const count = (pass: THREE.Mesh) => (pass.geometry as THREE.InstancedBufferGeometry).instanceCount
        expect(count(over)).toBeLessThan(count(under))
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
