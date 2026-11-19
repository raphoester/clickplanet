import {readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {
    type BorderLines,
    COLOUR,
    coarseTolerancesOf,
    createBorderLines,
    halfWidthOf,
    HIGHLIGHT_WIDTH,
    limbOf,
    MOST_SLACK,
    ORBIT_INK,
    ORBIT_WIDTH,
    OVER,
    pieceSides,
    PROVINCE_COLOUR,
    PROVINCE_DARK_FROM,
    PROVINCE_DARK_UNTIL,
    PROVINCE_DASH,
    PROVINCE_GAP,
    PROVINCE_ORBIT_INK,
    TOLERANCE,
    UNDER,
    WIDTH,
} from "./borderLines.ts"
import {BOARDS} from "./boardsAsset.ts"
import {CULL_SLACK} from "./chunkSight.ts"
import {EARTH_RADIUS} from "./sphere.ts"
import {flagPaint, flagsZoomOf, inOrbit, tilePointSize, tileSpacing, tilesZoomOf} from "./pointSize.ts"
import {maxZoomOf, MIN_ZOOM} from "./zoom.ts"
import {type BoardScale, EARTH_1_SCALE, scaleOf} from "../../domain/boardScale.ts"
import {type BorderLineData, decodeBorderLines, encodeBorderLines} from "../../domain/borderLinesFile.ts"
import {directionsOf, MOST_DIP, offsetOf} from "../../domain/coarseOutline.ts"
import {outlineSegments, outlineSetsOf, type ProvinceOutline, provinceOutlineOf} from "../../domain/outlineSets.ts"
import {decodeProvinceLines} from "../../domain/provinceLinesFile.ts"

const MAX_ZOOM = maxZoomOf(EARTH_1_SCALE)

const STATIC = fileURLToPath(new URL("../../../static/", import.meta.url))

function fileOf(board: string, name: string): ArrayBuffer {
    const bytes = readFileSync(`${STATIC}boards/${board}/${name}`)
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
}

// Every shipped board's outline, and its province lines if it has any, read once by the first test that needs them.
const SHIPPED = Object.values(BOARDS).map((board) => {
    let data: ReturnType<typeof decodeBorderLines> | undefined
    let provinces: ProvinceOutline | undefined
    const read = () => data ??= decodeBorderLines(fileOf(board.name, board.page.borderLines))
    const {provinceLines} = board.page
    // As the globe draws it once its province lines are read.
    const outline = () => {
        const lines = linesOf(read(), scaleOf(board))
        if (provinceLines) {
            lines.takeProvinces(provinces ??= provinceOutlineOf(decodeProvinceLines(fileOf(board.name, provinceLines)), coarseTolerancesOf(scaleOf(board))))
        }
        return lines
    }
    return {board, read, outline}
})

// As the globe makes it, from the cell edges and the board's coarse sets.
function linesOf(data: BorderLineData, scale: BoardScale): BorderLines {
    return createBorderLines(outlineSetsOf(data, coarseTolerancesOf(scale)), scale)
}

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

// One tile of a country among another's tiles. Its cell's sides are half a spacing from its centre and its corners a
// spacing over √3; its six neighbours are a spacing away, each across one side. The line runs along the sides, so the
// corridor between two discs is narrowest at a side's middle, and widest at a corner, where each piece reaches half a
// width past its end toward the third tile there.
describe("a lone tile's outline, at every zoom the tiles are drawn alone", () => {
    const SPACING = 20000
    const around = (reach: number, turn: number) => [0, 1, 2, 3, 4, 5].map((which) => {
        const angle = which * Math.PI / 3 + turn
        return [reach * Math.cos(angle), reach * Math.sin(angle)]
    })
    const corners = around(SPACING / Math.sqrt(3), 0)
    const cell = [...corners, corners[0]].flatMap(([x, y]) => [Math.round(x), Math.round(y), 0])
    const {from, to} = outlineSegments(decodeBorderLines(blobOf([cell])))

    // In CSS pixels on the screen's worst ratio, 1, from a disc's centre to the nearest pixel the line paints solid: the
    // vertex shader's quad runs half a width past each end of a piece, and its sides fade out past WIDTH.
    const clearance = ([x, y]: number[], perUnit: number) => {
        const reach = halfWidthOf(1)
        let nearest = Infinity
        for (let piece = 0; piece < from.length / 3; piece++) {
            const ax = from[piece * 3] * perUnit, ay = from[piece * 3 + 1] * perUnit
            const bx = to[piece * 3] * perUnit, by = to[piece * 3 + 1] * perUnit
            const length = Math.hypot(bx - ax, by - ay)
            const ux = (bx - ax) / length, uy = (by - ay) / length
            const along = (x * perUnit - ax) * ux + (y * perUnit - ay) * uy
            const across = Math.abs((y * perUnit - ay) * ux - (x * perUnit - ax) * uy)
            const past = Math.max(-reach - along, 0, along - length - reach)
            nearest = Math.min(nearest, Math.hypot(past, Math.max(0, across - WIDTH / 2)))
        }
        return nearest
    }

    const zooms: number[] = []
    for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
        if (flagPaint(zoom, 900, "split", EARTH_1_SCALE) === 0) zooms.push(zoom)
    }

    const expectClear = (discs: number[][]) => {
        expect(zooms.length).toBeGreaterThan(0)
        for (const zoom of zooms) {
            const perUnit = tileSpacing(zoom, 900, EARTH_1_SCALE) / SPACING
            const radius = tilePointSize(zoom, 900, EARTH_1_SCALE) / 2
            for (const disc of discs) expect(clearance(disc, perUnit), `zoom ${zoom}, disc at ${disc}`).toBeGreaterThan(radius)
        }
    }

    it("keeps the line off the tile's own disc, at the middle of each side", () => {
        expectClear([[0, 0]])
    })

    it("keeps each piece's end off the neighbours' discs, at the corners", () => {
        expectClear(around(SPACING, Math.PI / 6))
    })
})

describe("the two passes", () => {
    it("sit either side of the tiles, and above the earth", () => {
        const earth = EARTH_RADIUS
        expect(UNDER).toBeGreaterThan(earth)
        expect(UNDER).toBeLessThan(1)
        expect(OVER).toBeGreaterThan(1)
    })

    it("lift the over pass above the tiles five times as far as a coarse piece's middle dips", () => {
        expect(MOST_DIP * 5).toBeLessThanOrEqual(OVER - 1 + 1e-12)
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
        const {from} = outlineSegments(data)
        const sides = pieceSides(data)

        expect(sides.length).toBe(from.length / 3 * 2)
        expect([...sides]).toEqual([0, 4, 4, 9, 4, 9])
    })
})

describe("the highlight", () => {
    it("draws one landmass's outline over the tiles, and nothing when none is set", () => {
        const lines = linesOf(decodeBorderLines(blobOf([[1, 2, 3, 4, 5, 6, 7, 8, 9]], [3, 7])), EARTH_1_SCALE)
        const {group, material: {uniforms}} = passesOf(lines)[2]
        expect(group.visible).toBe(false)

        lines.setHighlight("countries", 7)
        expect([group.visible, uniforms.highlit.value, uniforms.lift.value]).toEqual([true, 7, OVER])

        lines.setHighlight("countries", 0)
        expect(group.visible).toBe(false)

        lines.dispose()
    })
})

describe("the handover", () => {
    const outline = () => {
        const lines = linesOf(decodeBorderLines(blobOf([[1, 2, 3, 4, 5, 6, 7, 8, 9]])), EARTH_1_SCALE)
        const [over, under] = passesOf(lines)
        const inkOf = ({group, material}: Pass) => group.visible ? material.uniforms.ink.value as number : 0
        return {lines, over, under, inkOf}
    }

    it("hands the outline from over the tiles to under them, and never drops it", () => {
        const {lines, over, under, inkOf} = outline()

        for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
            lines.update(zoom, 1600, 900, 1, "split", 0)

            const painted = flagPaint(zoom, 900, "split", EARTH_1_SCALE)
            expect(inkOf(over), `over at zoom ${zoom}`).toBe(painted * ORBIT_INK)
            expect(inkOf(under) + inkOf(over) / ORBIT_INK, `ink at zoom ${zoom}`).toBeGreaterThanOrEqual(1)
        }

        lines.dispose()
    })


    it("draws nothing it cannot see", () => {
        const {lines, over, under} = outline()

        lines.update(MIN_ZOOM, 1600, 900, 1, "split", 0)
        expect([over.group.visible, under.group.visible]).toEqual([true, false])

        lines.update(MAX_ZOOM, 1600, 900, 1, "split", 0)
        expect([over.group.visible, under.group.visible]).toEqual([false, true])

        lines.dispose()
    })

    it("keeps the outline under the tiles at every zoom when no flag is painted", () => {
        const {lines, over, under} = outline()

        for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
            lines.update(zoom, 1600, 900, 1, "tiles", 0)
            expect([over.group.visible, under.group.visible], `zoom ${zoom}`).toEqual([false, true])
        }

        lines.dispose()
    })

    it("keeps its width and its handover in CSS pixels on a sharper screen", () => {
        const {lines, over, under, inkOf} = outline()

        for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
            lines.update(zoom, 3200, 1800, 2, "split", 0)
            expect(inkOf(over), `over at zoom ${zoom}`).toBe(flagPaint(zoom, 900, "split", EARTH_1_SCALE) * ORBIT_INK)
        }

        expect(over.material.uniforms.halfWidth.value).toBe(halfWidthOf(2, ORBIT_WIDTH))
        expect(under.material.uniforms.halfWidth.value).toBe(halfWidthOf(2))

        expect(halfWidthOf(2) - halfWidthOf(1)).toBeCloseTo(WIDTH / 2)

        lines.dispose()
    })
})

const directions = new WeakMap<ArrayLike<number>, Float32Array>()

function unitsOf(ends: Int16Array): Float32Array {
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
        const from = unitsOf(endsOf("from")), to = unitsOf(endsOf("to"))
        const stacked = pass.material.defines.STACKED === true
        const standsFor = stacked ? (geometry.getAttribute("standsFor") as THREE.InterleavedBufferAttribute).data.array as Float32Array : undefined
        const pieces = from.length / 3

        const covered = new Uint8Array(pieces)
        let drawn = 0
        for (const {start, count} of drawnOf(pass)) {
            covered.fill(1, start, start + count)
            drawn += count
        }

        const {halfViewport, halfWidth, lift, limb} = pass.material.uniforms
        const hx = halfViewport.value.x, hy = halfViewport.value.y, w = halfWidth.value as number, l = lift.value as number
        const screenX = (ends: Float32Array, i: number) => ((m[0] * ends[i] + m[4] * ends[i + 1] + m[8] * ends[i + 2]) * l + m[12]) * hx
        const screenY = (ends: Float32Array, i: number) => ((m[1] * ends[i] + m[5] * ends[i + 1] + m[9] * ends[i + 2]) * l + m[13]) * hy

        let missed = 0
        for (let piece = 0; piece < pieces; piece++) {
            if (covered[piece]) continue
            const i = piece * 3
            const across = standsFor ? w / standsFor[piece * 2 + 1] : w
            const slack = limb.value + (across + 1) / Math.max(hy * p11, 1)
            const headFacing = from[i] * back.x + from[i + 1] * back.y + from[i + 2] * back.z
            const tailFacing = to[i] * back.x + to[i + 1] * back.y + to[i + 2] * back.z
            if (headFacing < -slack && tailFacing < -slack) continue

            const ax = screenX(from, i), ay = screenY(from, i), bx = screenX(to, i), by = screenY(to, i)
            const length = Math.hypot(bx - ax, by - ay)
            const ux = length > 1e-6 ? (bx - ax) / length : 1, uy = length > 1e-6 ? (by - ay) / length : 0
            const minX = Math.min(ax - ux * w, bx + ux * w) - Math.abs(uy) * across, maxX = Math.max(ax - ux * w, bx + ux * w) + Math.abs(uy) * across
            const minY = Math.min(ay - uy * w, by + uy * w) - Math.abs(ux) * across, maxY = Math.max(ay - uy * w, by + uy * w) + Math.abs(ux) * across
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

const PASS_NAMES = ["over", "under", "highlight", "mission", "province highlight", "provinces over", "provinces under", "provinces' highlight"]

describe.each(SHIPPED)("the cull of $board.name's outline", ({outline}) => {
    const look = (lines: BorderLines, direction: readonly number[], zoom: number, ratio: number, target?: readonly number[]) => {
        lines.update(zoom, WIDE * ratio, HIGH * ratio, ratio, "split", 0)
        const camera = cameraOf(direction, zoom, target)
        lines.cull(camera)
        return missedBy(lines, camera)
    }

    it("keeps every piece any pass would draw, from every camera, until it has turned by the slack", {timeout: 120_000}, () => {
        const lines = outline()
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
        const lines = outline()
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
        const lines = outline()
        const shareAt = (zoom: number) => look(lines, [1, 0.3, 0.2], zoom, 2).map(({drawn, pieces}) => drawn / pieces)
        expect(Math.max(...shareAt(10))).toBeLessThan(0.1)
        expect(Math.max(...shareAt(0.5))).toBeLessThan(0.9)
        lines.dispose()
    })

    it("draws the pieces in their order, pass after pass, the same pieces in every pass of the countries' cell edges", () => {
        const lines = outline()
        look(lines, [-0.4, 0.8, -0.1], 3, 2)
        const passes = passesOf(lines)

        const ids = passes.flatMap(({meshes}) => meshes.map(({id}) => id))
        expect(ids).toEqual([...ids].sort((a, b) => a - b))

        const cells = passes.filter((pass) => pass.meshes[0].geometry === passes[1].meshes[0].geometry)
        const [under, ...others] = [cells[1], cells[0], ...cells.slice(2)].map(drawnOf)
        expect(under.length).toBeGreaterThan(1)
        expect(under.length).toBeLessThanOrEqual(8)
        for (let i = 1; i < under.length; i++) expect(under[i].start).toBeGreaterThan(under[i - 1].start + under[i - 1].count)
        for (const pass of others) expect(pass).toEqual(under)
        lines.dispose()
    })
})

describe("the province lines", () => {
    // Two landmasses, 3 and 7, the second cut into provinces 11 and 12 by one line of two pieces.
    const outline = (sides = Uint16Array.from([5, 11, 5, 12])) => {
        const lines = linesOf(decodeBorderLines(blobOf([[1, 2, 3, 4, 5, 6, 7, 8, 9]], [3, 7])), EARTH_1_SCALE)
        lines.takeProvinces({
            countryLines: 0, provinces: 0, countrySides: sides,
            sets: outlineSetsOf(decodeBorderLines(blobOf([[10, 11, 12, 13, 14, 15, 16, 17, 18]], [11, 12])), coarseTolerancesOf(EARTH_1_SCALE)),
        })
        const [over, under, highlight, mission, countryHighlight, provinceOver, provinceUnder, provinceHighlight] = passesOf(lines)
        return {lines, over, under, highlight, mission, countryHighlight, provinceOver, provinceUnder, provinceHighlight}
    }

    it("draw from orbit only once the provinces divide the map, fading with their frames, and pushed in always", () => {
        const {lines, provinceOver, provinceUnder} = outline()
        for (const view of ["big", "split", "tiles"] as const) {
            for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
                const painted = flagPaint(zoom, 900, view, EARTH_1_SCALE)
                const orbit = inOrbit(zoom, 900, EARTH_1_SCALE)
                lines.update(zoom, 1600, 900, 1, view, 0)
                expect([provinceOver.group.visible, provinceUnder.group.visible], `${view} at zoom ${zoom}`).toEqual([false, !orbit && painted < 1])

                for (const share of [0.5, 1]) {
                    lines.update(zoom, 1600, 900, 1, view, share)
                    expect([provinceOver.group.visible, provinceUnder.group.visible], `${view} at zoom ${zoom}`).toEqual([painted > 0, painted < 1])
                    expect(provinceOver.material.uniforms.ink.value).toBe(painted * PROVINCE_ORBIT_INK * share)
                }
            }
        }
        lines.dispose()
    })

    it("darken under the tiles as they grow, to the countries' line and dashed once a tile is big", () => {
        const {lines, provinceUnder} = outline()
        const {uniforms} = provinceUnder.material
        let last = 0
        for (let zoom = tilesZoomOf(900, EARTH_1_SCALE); zoom <= MAX_ZOOM; zoom += 0.05) {
            lines.update(zoom, 1600, 900, 2, "split", 0)
            const tile = tilePointSize(zoom, 900 / 2, EARTH_1_SCALE)
            const colour = uniforms.colour.value as THREE.Color
            const dashed = uniforms.dashed.value as number
            expect(dashed, `zoom ${zoom}`).toBeGreaterThanOrEqual(last)
            last = dashed
            if (tile <= PROVINCE_DARK_FROM) expect([colour.r, dashed], `zoom ${zoom}`).toEqual([PROVINCE_COLOUR.r, 0])
            if (tile >= PROVINCE_DARK_UNTIL) expect([colour.r, dashed], `zoom ${zoom}`).toEqual([COLOUR.r, 1])
            expect(colour.r, `zoom ${zoom}`).toBeCloseTo(PROVINCE_COLOUR.r + (COLOUR.r - PROVINCE_COLOUR.r) * dashed, 6)
            // In drawing-buffer pixels, at a ratio of 2.
            expect([uniforms.dashPeriod.value, uniforms.dashHalf.value]).toEqual([(PROVINCE_DASH + PROVINCE_GAP) * 2, PROVINCE_DASH])
        }
        expect(last).toBe(1)
        expect(PROVINCE_COLOUR.r).toBe(0.6)
        lines.dispose()
    })

    it("keep the countries' lines as they are whatever divides the map", () => {
        const {lines, over, under} = outline()
        for (let zoom = MIN_ZOOM; zoom <= MAX_ZOOM; zoom += 0.05) {
            const seen = (share: number) => {
                lines.update(zoom, 1600, 900, 1, "split", share)
                return [over.group.visible, under.group.visible, over.material.uniforms.ink.value]
            }
            expect(seen(1), `zoom ${zoom}`).toEqual(seen(0))
            expect(seen(0.5), `zoom ${zoom}`).toEqual(seen(0))
        }
        lines.dispose()
    })

    it("light a province's whole outline, its province lines and its stretch of the coast, and no landmass", () => {
        const {lines, highlight, countryHighlight, provinceHighlight} = outline()

        lines.setHighlight("provinces", 12)
        expect([highlight.group.visible, countryHighlight.group.visible, provinceHighlight.group.visible]).toEqual([false, true, true])
        expect([countryHighlight.material.defines.PROVINCES, countryHighlight.material.uniforms.highlit.value]).toEqual([true, 12])
        expect(provinceHighlight.material.uniforms.highlit.value).toBe(12)

        lines.setHighlight("countries", 7)
        expect([highlight.group.visible, countryHighlight.group.visible, provinceHighlight.group.visible]).toEqual([true, false, false])

        lines.setHighlight("provinces", 0)
        expect([highlight.group.visible, countryHighlight.group.visible, provinceHighlight.group.visible]).toEqual([false, false, false])
        lines.dispose()
    })

    it("light a province asked for before its lines were read once they are", () => {
        const lines = linesOf(decodeBorderLines(blobOf([[1, 2, 3, 4, 5, 6, 7, 8, 9]], [3, 7])), EARTH_1_SCALE)
        lines.setHighlight("provinces", 12)
        lines.takeProvinces({
            countryLines: 0, provinces: 0, countrySides: Uint16Array.from([5, 11, 5, 12]),
            sets: outlineSetsOf(decodeBorderLines(blobOf([[10, 11, 12, 13, 14, 15]], [11, 12])), coarseTolerancesOf(EARTH_1_SCALE)),
        })
        const [, , highlight, , countryHighlight, , , provinceHighlight] = passesOf(lines)
        expect([highlight.group.visible, countryHighlight.group.visible, provinceHighlight.group.visible]).toEqual([false, true, true])
        lines.dispose()
    })

    it("light a province's stretch of the coast on the cell edges, each with its provinces from its range's first piece", () => {
        const {lines, countryHighlight} = outline()
        lines.update(MIN_ZOOM, 1600, 900, 1, "split", 1)
        lines.cull(cameraOf([0, 0, 1], MIN_ZOOM))
        for (const mesh of countryHighlight.meshes) {
            const geometry = mesh.geometry as THREE.InstancedBufferGeometry
            const from = geometry.getAttribute("from") as THREE.InterleavedBufferAttribute
            const provinces = geometry.getAttribute("provinces") as THREE.InterleavedBufferAttribute
            expect(provinces.offset).toBe(from.offset / 3 * 2)
            expect(Array.from(provinces.data.array as Uint16Array)).toEqual([5, 11, 5, 12])
            expect(geometry.getAttribute("standsFor")).toBeUndefined()
        }
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
        const lines = linesOf(decodeBorderLines(blobOf(runs)), EARTH_1_SCALE)

        const pan = Math.min(CULL_SLACK / zoom, MOST_SLACK) * 0.9999
        lines.update(zoom, WIDE * ratio, HIGH * ratio, ratio, "split", 0)
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

type Ends = {from: Int16Array, to: Int16Array}

function endsOf({meshes}: Pass): Ends {
    const geometry = meshes[0].geometry as THREE.InstancedBufferGeometry
    const array = (name: string) => (geometry.getAttribute(name) as THREE.InterleavedBufferAttribute).data.array as Int16Array
    return {from: array("from"), to: array("to")}
}

const deviations = new WeakMap<Int16Array, number>()

// The farthest a set's pieces lie from the cell edges they stand for, on the globe of radius 1: each piece runs from a
// cell edges' corner to a later one of the same run, and the corners between are the ones it dropped.
function deviationOf(coarse: Ends, cells: Ends): number {
    const known = deviations.get(coarse.from)
    if (known !== undefined) return known
    const same = (a: Int16Array, i: number, b: Int16Array, j: number) => [0, 1, 2].every((axis) => a[i * 3 + axis] === b[j * 3 + axis])
    const cellTo = directionsOf(cells.to), coarseFrom = directionsOf(coarse.from), coarseTo = directionsOf(coarse.to)
    const three = new Float64Array(9)
    let farthest = 0, cell = 0
    for (let piece = 0; piece < coarse.from.length / 3; piece++) {
        expect(same(cells.from, cell, coarse.from, piece), `piece ${piece} starts on a cell edge's corner`).toBe(true)
        three.set(coarseFrom.subarray(piece * 3, piece * 3 + 3), 0)
        three.set(coarseTo.subarray(piece * 3, piece * 3 + 3), 3)
        for (; !same(cells.to, cell, coarse.to, piece); cell++) {
            three.set(cellTo.subarray(cell * 3, cell * 3 + 3), 6)
            farthest = Math.max(farthest, offsetOf(three, 2, 0, 1))
        }
        cell++
    }
    expect(cell).toBe(cells.from.length / 3)
    deviations.set(coarse.from, farthest)
    return farthest
}

describe.each(SHIPPED)("the over and the highlight pass of $board.name's outline", ({board, read}) => {
    it("stay within the tolerance of the cell edges at every zoom they are drawn at, on every screen", {timeout: 120_000}, () => {
        const scale = scaleOf(board)
        const lines = linesOf(read(), scale)
        const cells = outlineSegments(read())
        // CSS pixels wide and high, and the ratio.
        const screens = [[390, 640, 2], [430, 932, 2], [1280, 800, 2], [1920, 1000, 1], [1280, 1400, 1.5]]
        let coarse = 0
        for (const [wide, high, ratio] of screens) {
            for (let zoom = MIN_ZOOM; zoom < tilesZoomOf(high, scale); zoom *= 1.04) {
                lines.update(zoom, wide * ratio, high * ratio, ratio, "split", 0)
                lines.cull(cameraOf([1, 0.3, 0.2], zoom))
                const [over, , highlight] = passesOf(lines)
                const drawn = endsOf(over)
                expect(over.group.visible, `zoom ${zoom}`).toBe(true)
                expect(endsOf(highlight).from).toBe(drawn.from)
                if (drawn.from.length === cells.from.length) continue
                coarse++
                const perUnit = high * ratio / 2 * zoom
                expect(deviationOf(drawn, cells) * OVER * perUnit, `zoom ${zoom} on ${wide}×${high} at ${ratio}`).toBeLessThanOrEqual(TOLERANCE)
            }
            expect(flagsZoomOf(high, scale)).toBeLessThan(tilesZoomOf(high, scale))
        }
        if (board.name !== "siege-1") expect(coarse).toBeGreaterThan(0)
        lines.dispose()
    })
})

describe("the outline of earth-2 from orbit", () => {
    const shipped = SHIPPED.find(({board}) => board.name === "earth-2")!

    it("draws under a quarter of its pieces at the landing view over the flags, and the cell edges under them", () => {
        const lines = linesOf(shipped.read(), scaleOf(shipped.board))
        const cells = outlineSegments(shipped.read()).from.length / 3
        const piecesAt = (zoom: number) => {
            lines.update(zoom, 1280 * 2, 800 * 2, 2, "split", 0)
            lines.cull(cameraOf([1, 0.3, 0.2], zoom))
            return passesOf(lines).map((pass) => endsOf(pass).from.length / 3)
        }

        const [over, under, highlight, mission] = piecesAt(0.5)
        expect(over / cells).toBeLessThan(0.25)
        expect(highlight).toBe(over)
        expect([under, mission]).toEqual([cells, cells])

        expect(piecesAt(2)).toEqual([cells, cells, cells, cells])
        lines.dispose()
    })
})

// The vertex and the fragment shader's ink, copied, on screen in pixels: a run of cell edges along x, drawn as its cell
// edges and as pieces through every few corners. A coarse piece inks as its cell edges' quads laid over one another.
describe("the stacked ink", () => {
    type Quad = {ax: number, ay: number, ux: number, uy: number, length: number, edges: number, slant: number}

    // Each cell edge 30° across the run, or along it and 60° across in turn.
    const zigzag = (edge: number) => Array.from({length: 121}, (_, at) => [at * edge * Math.cos(Math.PI / 6), at % 2 ? edge / 2 : 0])
    const armchair = (edge: number) => {
        const corners = [[0, 0]]
        for (let at = 0; at < 120; at++) {
            const [x, y] = corners[at], turn = [0, Math.PI / 3, 0, -Math.PI / 3][at % 4]
            corners.push([x + edge * Math.cos(turn), y + edge * Math.sin(turn)])
        }
        return corners
    }

    const quadsOf = (corners: number[][], every: number): Quad[] => {
        const quads: Quad[] = []
        for (let at = 0; at + every < corners.length; at += every) {
            let path = 0
            for (let corner = at; corner < at + every; corner++) path += Math.hypot(corners[corner + 1][0] - corners[corner][0], corners[corner + 1][1] - corners[corner][1])
            const [ax, ay] = corners[at], [bx, by] = corners[at + every]
            const length = Math.hypot(bx - ax, by - ay)
            quads.push({ax, ay, ux: (bx - ax) / length, uy: (by - ay) / length, length, edges: every, slant: length / path})
        }
        return quads
    }

    // The line's ink across, averaged along its middle third: the integral of its opacity, in pixels.
    const inkOf = (quads: Quad[], halfWidth: number, ink: number, stacked: boolean, x0: number, x1: number) => {
        let total = 0, samples = 0
        for (let x = x0; x < x1; x += 0.1) {
            for (let y = -3 * halfWidth - 1; y <= 3 * halfWidth + 1; y += 0.1) {
                let clear = 1
                for (const {ax, ay, ux, uy, length, edges, slant} of quads) {
                    const along = (x - ax) * ux + (y - ay) * uy, across = (y - ay) * ux - (x - ax) * uy
                    const wide = stacked ? halfWidth / slant : halfWidth
                    if (along < -halfWidth || along > length + halfWidth || Math.abs(across) > wide) continue
                    let alpha = Math.min(1, Math.max(0, halfWidth - Math.abs(across) * halfWidth / wide)) * ink
                    if (stacked) {
                        const layers = Math.min(1 + 2 * halfWidth * edges * slant / Math.max(length, 1e-3), 64)
                        const covering = 1 + (along <= halfWidth ? 1 : 0) + (along >= length - halfWidth ? 1 : 0)
                        alpha = 1 - (1 - alpha) ** (layers / covering)
                    }
                    clear *= 1 - alpha
                }
                total += 1 - clear
            }
            samples++
        }
        return total * 0.1 / samples
    }

    const lines = [
        {name: "over, at a ratio of 1", halfWidth: halfWidthOf(1, ORBIT_WIDTH), ink: ORBIT_INK},
        {name: "over, at a ratio of 2", halfWidth: halfWidthOf(2, ORBIT_WIDTH), ink: ORBIT_INK},
        {name: "highlight, at a ratio of 2", halfWidth: halfWidthOf(2, HIGHLIGHT_WIDTH), ink: 1},
    ]
    const cases = [{shape: "zigzag", run: zigzag}, {shape: "armchair", run: armchair}].flatMap((shape) => lines.map((line) => ({...shape, ...line})))

    // The cell edges a coarse set's pieces are drawn over: under a pixel each.
    it.each(cases)("inks the $name line as its cell edges did, $shape", ({run, halfWidth, ink}) => {
        for (const edge of [0.25, 0.5, 1]) {
            const corners = run(edge)
            const x0 = corners[40][0], x1 = corners[80][0]
            const cells = inkOf(quadsOf(corners, 1), halfWidth, ink, false, x0, x1)
            for (const every of [4, 16]) {
                const ratio = inkOf(quadsOf(corners, every), halfWidth, ink, true, x0, x1) / cells
                expect(ratio, `cell edges of ${edge}px, pieces of ${every}`).toBeGreaterThan(0.85)
                expect(ratio, `cell edges of ${edge}px, pieces of ${every}`).toBeLessThan(1.15)
            }
        }
    })

    it("would ink the line a third lighter from orbit, each pixel once", () => {
        const {halfWidth, ink} = lines[1]
        const corners = zigzag(0.5)
        const x0 = corners[40][0], x1 = corners[80][0]
        const cells = inkOf(quadsOf(corners, 1), halfWidth, ink, false, x0, x1)
        expect(inkOf(quadsOf(corners, 4), halfWidth, ink, false, x0, x1) / cells).toBeLessThan(0.7)
    })
})

// The fragment shader's dashOf (shaders/borderLine/fragment.glsl), copied: how much of a pixel `along` a piece of
// `length` drawing-buffer pixels its dashes ink.
function dashOf(along: number, length: number, period: number, half: number): number {
    if (length < 1) return 1
    const count = Math.max(1, Math.floor(length / period + 0.5))
    const step = length / count
    const dash = half * (count > 1 ? step : Math.max(step, period)) / period
    const t = along / step
    return Math.min(1, Math.max(0, dash + 0.5 - Math.abs(t - Math.floor(t + 0.5)) * step))
}

describe("the province line's dashes", () => {
    const period = PROVINCE_DASH + PROVINCE_GAP, half = PROVINCE_DASH / 2

    it("ink both ends of every piece, so a line turning from one cell edge to the next is drawn at the turn", () => {
        for (let length = 1; length <= 120; length += 0.5) {
            expect(dashOf(0, length, period, half), `${length}px`).toBe(1)
            expect(dashOf(length, length, period, half), `${length}px`).toBe(1)
        }
    })

    it("leave a gap in every piece a dash and a gap long, as many as its whole periods", () => {
        for (let length = period; length <= 120; length += 0.5) {
            const gaps = Math.max(1, Math.floor(length / period + 0.5))
            let found = 0
            for (let along = 0, inGap = false; along <= length; along += 0.05) {
                const gap = dashOf(along, length, period, half) === 0
                if (gap && !inGap) found++
                inGap = gap
            }
            expect(found, `${length}px`).toBe(gaps)
        }
    })

    it("ink about as much of a long line as a dash is of a period", () => {
        let ink = 0
        const length = 100 * period
        for (let along = 0; along < length; along += 0.05) ink += dashOf(along, length, period, half) * 0.05
        expect(ink / length).toBeCloseTo(PROVINCE_DASH / period, 1)
    })
})
