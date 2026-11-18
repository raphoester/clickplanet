import {readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {
    type BorderLines,
    COARSE_SAMPLES,
    createBorderLines,
    halfWidthOf,
    limbOf,
    MOST_SLACK,
    ORBIT_INK,
    ORBIT_WIDTH,
    outlineChunksOf,
    outlineSegments,
    OVER,
    pieceSides,
    SAMPLES,
    UNDER,
    WIDTH,
} from "./borderLines.ts"
import {BOARDS} from "./boardsAsset.ts"
import {CULL_SLACK} from "./chunkSight.ts"
import {EARTH_RADIUS} from "./sphere.ts"
import {flagPaint, tilePointSize, tileSpacing} from "./pointSize.ts"
import {maxZoomOf, MIN_ZOOM} from "./zoom.ts"
import {EARTH_1_SCALE, scaleOf} from "../../domain/boardScale.ts"
import {decodeBorderLines, encodeBorderLines} from "../../domain/borderLinesFile.ts"
import {CAP_FLOATS, chunkCount} from "../../domain/tileChunks.ts"

const MAX_ZOOM = maxZoomOf(EARTH_1_SCALE)

const STATIC = fileURLToPath(new URL("../../../static/", import.meta.url))

// Every shipped board's outline, read once by the first test that needs it.
const SHIPPED = Object.values(BOARDS).map((board) => {
    let data: ReturnType<typeof decodeBorderLines> | undefined
    const read = () => {
        if (data) return data
        const bytes = readFileSync(`${STATIC}boards/${board.name}/${board.page.borderLines}`)
        return data = decodeBorderLines(bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength))
    }
    return {board, read}
})

type Pass = {group: THREE.Group, material: THREE.ShaderMaterial, meshes: THREE.Mesh[]}

// over, under, highlight and mission, each a group of the draws that share its material.
function passesOf(lines: BorderLines): Pass[] {
    return (lines.object.children as THREE.Group[]).map((group) => {
        const meshes = group.children as THREE.Mesh[]
        return {group, meshes, material: meshes[0].material as THREE.ShaderMaterial}
    })
}

function drawnOf({meshes}: Pass): {start: number, count: number}[] {
    return meshes.filter((mesh) => mesh.visible).map((mesh) => {
        const geometry = mesh.geometry as THREE.InstancedBufferGeometry
        return {start: (geometry.getAttribute("from") as THREE.InterleavedBufferAttribute).offset / 3, count: geometry.instanceCount}
    })
}

function blobOf(runs: number[][], sides: number[] = runs.flatMap(() => [0, 1])): ArrayBuffer {
    const bytes = encodeBorderLines({
        runs: Uint32Array.from(runs, (run) => run.length / 3),
        sides: Uint16Array.from(sides),
        corners: Int16Array.from(runs.flat()),
    })
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
}

describe("outlineSegments", () => {
    const at = (points: Int16Array, piece: number) => [...points.slice(piece * 3, piece * 3 + 3)]

    it("starts and ends a run exactly on its own corners", () => {
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

        expect(at(to, 2 * samples - 1)).toEqual([0, 10000, 30000])
        expect(at(from, 2 * samples)).toEqual([30000, 0, 0])
    })

    const SPACING = 20000
    const cell = () => [0, 1, 2, 3, 4, 5, 0].flatMap((which) => {
        const angle = which * Math.PI / 3
        const reach = SPACING / Math.sqrt(3)
        return [Math.round(reach * Math.cos(angle)), Math.round(reach * Math.sin(angle)), 0]
    })

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

    it("keeps the line clear of the tiles it runs between, at every zoom it is drawn alone", () => {
        const share = nearest(SAMPLES) / SPACING

        for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
            if (flagPaint(zoom, 900, "split", EARTH_1_SCALE) > 0) continue
            const clearance = share * tileSpacing(zoom, 900, EARTH_1_SCALE) - WIDTH / 2
            expect(clearance, `zoom ${zoom}`).toBeGreaterThan(tilePointSize(zoom, 900, EARTH_1_SCALE) / 2)
        }
    })

    it("draws the coarse pass close enough to the fine one to cross-fade with it", () => {
        expect(Math.abs(nearest(COARSE_SAMPLES) - nearest(SAMPLES))).toBeLessThan(SPACING * 0.02)
    })
})

describe("the two passes", () => {
    it("sit either side of the tiles, and above the earth", () => {
        const earth = EARTH_RADIUS
        expect(UNDER).toBeGreaterThan(earth)
        expect(UNDER).toBeLessThan(1)
        expect(OVER).toBeGreaterThan(1)
    })

    it("say how far past the limb each pass can still be seen", () => {
        const earth = EARTH_RADIUS

        for (const lift of [OVER, UNDER]) {
            const past = Math.asin(limbOf(lift))
            expect(lift * Math.cos(past)).toBeCloseTo(Math.min(earth, lift), 6)
        }

        expect(limbOf(OVER)).toBeGreaterThan(limbOf(UNDER))

        expect(limbOf(earth / 2)).toBe(0)
    })
})

describe("pieceSides", () => {
    it("gives every piece of a run the two landmasses the run is between", () => {
        const data = decodeBorderLines(blobOf([[1000, 0, 0, 1000, 500, 0], [0, 1000, 0, 0, 1000, 500, 0, 900, 900]], [0, 4, 4, 9]))
        const {from} = outlineSegments(data, 2)
        const sides = pieceSides(data, 2)

        expect(sides.length).toBe(from.length / 3 * 2)
        expect([...sides]).toEqual([...Array(4).fill([0, 4]), ...Array(6).fill([4, 9])].flat())
    })
})

describe("the highlight", () => {
    it("draws one landmass's outline over the tiles, and nothing when none is set", () => {
        const lines = createBorderLines(decodeBorderLines(blobOf([[1, 2, 3, 4, 5, 6, 7, 8, 9]], [3, 7])), EARTH_1_SCALE)
        const {group, material: {uniforms}} = passesOf(lines)[2]
        expect(group.visible).toBe(false)

        lines.setHighlight(7)
        expect([group.visible, uniforms.landmass.value, uniforms.lift.value]).toEqual([true, 7, OVER])

        lines.setHighlight(0)
        expect(group.visible).toBe(false)

        lines.dispose()
    })
})

describe("the handover", () => {
    const outline = () => {
        const lines = createBorderLines(decodeBorderLines(blobOf([[1, 2, 3, 4, 5, 6, 7, 8, 9]])), EARTH_1_SCALE)
        const [over, under] = passesOf(lines)
        const inkOf = ({group, material}: Pass) => group.visible ? material.uniforms.ink.value as number : 0
        return {lines, over, under, inkOf}
    }

    it("hands the outline from over the tiles to under them, and never drops it", () => {
        const {lines, over, under, inkOf} = outline()

        for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
            lines.update(zoom, 1600, 900, 1, "split")

            const painted = flagPaint(zoom, 900, "split", EARTH_1_SCALE)
            expect(inkOf(over), `over at zoom ${zoom}`).toBe(painted * ORBIT_INK)
            expect(inkOf(under) + inkOf(over) / ORBIT_INK, `ink at zoom ${zoom}`).toBeGreaterThanOrEqual(1)
        }

        lines.dispose()
    })


    it("draws nothing it cannot see", () => {
        const {lines, over, under} = outline()

        lines.update(MIN_ZOOM, 1600, 900, 1, "split")
        expect([over.group.visible, under.group.visible]).toEqual([true, false])

        lines.update(MAX_ZOOM, 1600, 900, 1, "split")
        expect([over.group.visible, under.group.visible]).toEqual([false, true])

        lines.dispose()
    })

    it("keeps the outline under the tiles at every zoom when no flag is painted", () => {
        const {lines, over, under} = outline()

        for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
            lines.update(zoom, 1600, 900, 1, "tiles")
            expect([over.group.visible, under.group.visible], `zoom ${zoom}`).toEqual([false, true])
        }

        lines.dispose()
    })

    it("gives the coarse outline to the pass that is drawn zoomed out", () => {
        const {lines, over, under} = outline()
        const count = (pass: Pass) => drawnOf(pass).reduce((sum, {count}) => sum + count, 0)
        expect(count(over)).toBeLessThan(count(under))
        lines.dispose()
    })

    it("keeps its width and its handover in CSS pixels on a sharper screen", () => {
        const {lines, over, under, inkOf} = outline()

        for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
            lines.update(zoom, 3200, 1800, 2, "split")
            expect(inkOf(over), `over at zoom ${zoom}`).toBe(flagPaint(zoom, 900, "split", EARTH_1_SCALE) * ORBIT_INK)
        }

        expect(over.material.uniforms.halfWidth.value).toBe(halfWidthOf(2, ORBIT_WIDTH))
        expect(under.material.uniforms.halfWidth.value).toBe(halfWidthOf(2))

        expect(halfWidthOf(2) - halfWidthOf(1)).toBeCloseTo(WIDTH / 2)

        lines.dispose()
    })
})

describe.each(SHIPPED)("the chunks of $board.name's outline", ({read}) => {
    it("hold every corner once, in order, and each cap every end of its corners' pieces at both resolutions", () => {
        const data = read()
        const fine = outlineSegments(data, SAMPLES)
        const chunks = outlineChunksOf(fine)
        const count = chunkCount(chunks)
        expect(chunks.starts[0]).toBe(0)
        expect(chunks.starts[count]).toBe(fine.from.length / 3 / SAMPLES)
        expect(SAMPLES % COARSE_SAMPLES).toBe(0)

        for (const samples of [COARSE_SAMPLES, SAMPLES]) {
            const {from, to} = outlineSegments(data, samples)
            let outside = 0
            for (let chunk = 0; chunk < count; chunk++) {
                const [x, y, z, radius] = chunks.caps.subarray(chunk * CAP_FLOATS, chunk * CAP_FLOATS + 4)
                for (let piece = chunks.starts[chunk] * samples; piece < chunks.starts[chunk + 1] * samples; piece++) {
                    for (const ends of [from, to]) {
                        const px = ends[piece * 3], py = ends[piece * 3 + 1], pz = ends[piece * 3 + 2]
                        const across = Math.hypot(py * z - pz * y, pz * x - px * z, px * y - py * x)
                        if (Math.atan2(across, px * x + py * y + pz * z) > radius) outside++
                    }
                }
            }
            expect(outside, `${samples} samples`).toBe(0)
        }
    })
})

const directions = new WeakMap<ArrayLike<number>, Float32Array>()

function directionsOf(ends: Int16Array): Float32Array {
    let found = directions.get(ends)
    if (found) return found
    found = new Float32Array(ends.length)
    for (let i = 0; i < ends.length; i += 3) {
        const length = Math.hypot(ends[i], ends[i + 1], ends[i + 2])
        found.set([ends[i] / length, ends[i + 1] / length, ends[i + 2] / length], i)
    }
    directions.set(ends, found)
    return found
}

// Each pass's pieces its vertex shader (shaders/borderLine/vertex.glsl) would lay out on screen and the cull left out.
function missedBy(lines: BorderLines, camera: THREE.OrthographicCamera): {missed: number, drawn: number, pieces: number}[] {
    const m = new THREE.Matrix4().multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse).elements
    const back = new THREE.Vector3().setFromMatrixColumn(camera.matrixWorld, 2).normalize()
    const p11 = camera.projectionMatrix.elements[5]

    return passesOf(lines).map((pass) => {
        const geometry = pass.meshes[0].geometry as THREE.InstancedBufferGeometry
        const endsOf = (name: string) => (geometry.getAttribute(name) as THREE.InterleavedBufferAttribute).data.array as Int16Array
        const from = directionsOf(endsOf("from")), to = directionsOf(endsOf("to"))
        const pieces = from.length / 3

        const covered = new Uint8Array(pieces)
        let drawn = 0
        for (const {start, count} of drawnOf(pass)) {
            covered.fill(1, start, start + count)
            drawn += count
        }

        const {halfViewport, halfWidth, lift, limb} = pass.material.uniforms
        const hx = halfViewport.value.x, hy = halfViewport.value.y, w = halfWidth.value as number, l = lift.value as number
        const slack = limb.value + (w + 1) / Math.max(hy * p11, 1)
        const screenX = (ends: Float32Array, i: number) => ((m[0] * ends[i] + m[4] * ends[i + 1] + m[8] * ends[i + 2]) * l + m[12]) * hx
        const screenY = (ends: Float32Array, i: number) => ((m[1] * ends[i] + m[5] * ends[i + 1] + m[9] * ends[i + 2]) * l + m[13]) * hy

        let missed = 0
        for (let piece = 0; piece < pieces; piece++) {
            if (covered[piece]) continue
            const i = piece * 3
            const headFacing = from[i] * back.x + from[i + 1] * back.y + from[i + 2] * back.z
            const tailFacing = to[i] * back.x + to[i + 1] * back.y + to[i + 2] * back.z
            if (headFacing < -slack && tailFacing < -slack) continue

            const ax = screenX(from, i), ay = screenY(from, i), bx = screenX(to, i), by = screenY(to, i)
            const length = Math.hypot(bx - ax, by - ay)
            const ux = length > 1e-6 ? (bx - ax) / length : 1, uy = length > 1e-6 ? (by - ay) / length : 0
            const minX = Math.min(ax - ux * w, bx + ux * w) - Math.abs(uy) * w, maxX = Math.max(ax - ux * w, bx + ux * w) + Math.abs(uy) * w
            const minY = Math.min(ay - uy * w, by + uy * w) - Math.abs(ux) * w, maxY = Math.max(ay - uy * w, by + uy * w) + Math.abs(ux) * w
            if (maxX >= -hx && minX <= hx && maxY >= -hy && minY <= hy) missed++
        }
        return {missed, drawn, pieces}
    })
}

const WIDE = 1280, HIGH = 800

function cameraOf(direction: readonly number[], zoom: number, target: readonly number[] = [0, 0, 0]): THREE.OrthographicCamera {
    const camera = new THREE.OrthographicCamera(-WIDE / HIGH, WIDE / HIGH, 1, -1, 0.01, 100)
    const at = new THREE.Vector3(...target)
    camera.position.set(direction[0], direction[1], direction[2]).setLength(5).add(at)
    camera.lookAt(at)
    camera.zoom = zoom
    camera.updateProjectionMatrix()
    camera.updateMatrixWorld()
    return camera
}

const PASS_NAMES = ["over", "under", "highlight", "mission"]

describe.each(SHIPPED)("the cull of $board.name's outline", ({board, read}) => {
    const look = (lines: BorderLines, direction: readonly number[], zoom: number, ratio: number, target?: readonly number[]) => {
        lines.update(zoom, WIDE * ratio, HIGH * ratio, ratio, "split")
        const camera = cameraOf(direction, zoom, target)
        lines.cull(camera)
        return missedBy(lines, camera)
    }

    it("keeps every piece any pass would draw, from every camera, until it has turned by the slack", {timeout: 120_000}, () => {
        const lines = createBorderLines(read(), scaleOf(board))
        const directions = [[1, 0.3, 0.2], [-0.4, 0.8, -0.1], [0.1, -0.9, 0.5], [-0.7, -0.2, -0.7]]
        directions.forEach((direction, d) => [40, 10, 3, 1, 0.5].forEach((zoom, z) => {
            const ratio = 1 + (d + z) % 2
            const found = cameraOf(direction, zoom)
            const right = new THREE.Vector3().setFromMatrixColumn(found.matrixWorld, 0)
            const turn = Math.min(CULL_SLACK / zoom, MOST_SLACK) * 0.98
            const turned = [
                {name: "found", axis: right, angle: 0},
                {name: "left", axis: new THREE.Vector3(0, 1, 0), angle: turn},
                {name: "right", axis: new THREE.Vector3(0, 1, 0), angle: -turn},
                {name: "up", axis: right, angle: turn},
                {name: "down", axis: right, angle: -turn},
            ]
            for (const {name, axis, angle} of turned) {
                const {x, y, z} = new THREE.Vector3(...direction).applyAxisAngle(axis, angle)
                look(lines, [x, y, z], zoom, ratio).forEach(({missed}, pass) => {
                    expect(missed, `${PASS_NAMES[pass]} at zoom ${zoom} from ${direction} turned ${name}, ratio ${ratio}`).toBe(0)
                })
            }
        }))
        lines.dispose()
    })

    it("keeps them through a turn, a zoom in and a pan, finding them again once the view has moved too far", {timeout: 120_000}, () => {
        const lines = createBorderLines(read(), scaleOf(board))
        for (const start of [1, 10]) {
            for (let step = 0; step < 14; step++) {
                const angle = Math.min(step, 8) * 0.006
                const zoom = start * (1 + 0.02 * Math.max(0, Math.min(step, 10) - 8))
                const target = [0.003 * Math.max(0, step - 10), 0, 0]
                look(lines, [Math.cos(angle), 0.3, Math.sin(angle)], zoom, 2, target).forEach(({missed}, pass) => {
                    expect(missed, `${PASS_NAMES[pass]} at step ${step} from zoom ${start}`).toBe(0)
                })
            }
        }
        lines.dispose()
    })

    it("draws a small share of the outline pushed in, and less than all of it from orbit", {timeout: 60_000}, () => {
        const lines = createBorderLines(read(), scaleOf(board))
        const shareAt = (zoom: number) => look(lines, [1, 0.3, 0.2], zoom, 2).map(({drawn, pieces}) => drawn / pieces)
        expect(Math.max(...shareAt(10))).toBeLessThan(0.1)
        expect(Math.max(...shareAt(0.5))).toBeLessThan(0.9)
        lines.dispose()
    })

    it("draws the pieces in their order, pass after pass, the same corners at both resolutions", () => {
        const lines = createBorderLines(read(), scaleOf(board))
        look(lines, [-0.4, 0.8, -0.1], 3, 2)
        const passes = passesOf(lines)

        const ids = passes.flatMap(({meshes}) => meshes.map(({id}) => id))
        expect(ids).toEqual([...ids].sort((a, b) => a - b))

        const [over, under] = passes.map(drawnOf)
        expect(under.length).toBeGreaterThan(1)
        expect(under.length).toBeLessThanOrEqual(8)
        for (let i = 1; i < under.length; i++) expect(under[i].start).toBeGreaterThan(under[i - 1].start + under[i - 1].count)
        const ratio = SAMPLES / COARSE_SAMPLES
        expect(over.map(({start, count}) => ({start: start * ratio, count: count * ratio}))).toEqual(under)
        lines.dispose()
    })
})

describe("the cull at the screen's edge", () => {
    it("keeps a line just off screen whose width reaches onto it, until the camera has panned by the slack", () => {
        const zoom = 10, ratio = 2
        const perUnit = HIGH * ratio / 2 * zoom
        const edge = WIDE / HIGH / zoom
        const point = (x: number, y: number) => [x, y, Math.sqrt(1 - x * x - y * y)].map((v) => Math.round(v * 32767))

        // Short and slanted, so the quad's corner reaches furthest; each line its own chunk, each 0.25px further out.
        const runs: number[][] = []
        for (let line = 0; line < 80; line++) {
            const x = edge + line * 0.25 / perUnit, y = -0.09 + line * 0.0022
            runs.push([...point(x, y), ...point(x + 0.00005, y + 0.00005), ...point(x + 0.0001, y + 0.0001)])
            runs.push([...point(0.1, line * 0.01), ...point(0.1, line * 0.01 + 0.002)].map((v, i) => i % 3 === 2 ? -v : v))
        }
        const lines = createBorderLines(decodeBorderLines(blobOf(runs)), EARTH_1_SCALE)

        const pan = Math.min(CULL_SLACK / zoom, MOST_SLACK) * 0.9999
        lines.update(zoom, WIDE * ratio, HIGH * ratio, ratio, "split")
        lines.cull(cameraOf([0, 0, 1], zoom, [-pan, 0, 0]))
        const found = drawnOf(passesOf(lines)[1])

        const shown = cameraOf([0, 0, 1], zoom)
        lines.cull(shown)
        expect(drawnOf(passesOf(lines)[1])).toEqual(found)
        missedBy(lines, shown).forEach(({missed, drawn, pieces}, pass) => {
            expect(drawn, PASS_NAMES[pass]).toBeLessThan(pieces)
            expect(missed, PASS_NAMES[pass]).toBe(0)
        })
        lines.dispose()
    })
})
