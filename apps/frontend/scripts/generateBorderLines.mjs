import fs from "node:fs"
import {createHash} from "node:crypto"
import * as THREE from "three"

const DETAIL = 300

const say = (...args) => console.error(...args)

say(`rebuilding the icosahedron at detail ${DETAIL}...`)
const raw = new THREE.IcosahedronGeometry(1, DETAIL).attributes.position.array

const vertexAt = new Map()
const px = [], py = [], pz = []
const corners = new Int32Array(raw.length / 3)
for (let i = 0; i < raw.length; i += 3) {
    const key = `${raw[i].toFixed(6)},${raw[i + 1].toFixed(6)},${raw[i + 2].toFixed(6)}`
    let v = vertexAt.get(key)
    if (v === undefined) {
        v = px.length
        vertexAt.set(key, v)
        px.push(raw[i]); py.push(raw[i + 1]); pz.push(raw[i + 2])
    }
    corners[i / 3] = v
}
const vertices = px.length
const faces = corners.length / 3
say(`lattice: ${vertices} vertices, ${faces} triangles`)

const coordinatesName = only("static", /^coordinates-[0-9a-f]{8}\.bin$/)
const coordinates = fs.readFileSync(`static/${coordinatesName}`)
const tiles = new DataView(coordinates.buffer, coordinates.byteOffset, coordinates.byteLength).getUint32(8, true)
const tilePositions = new Float32Array(coordinates.buffer, coordinates.byteOffset + 12, tiles * 3)

const vertexOfTile = new Int32Array(tiles)
for (let t = 0; t < tiles; t++) {
    const key = `${tilePositions[t * 3].toFixed(6)},${tilePositions[t * 3 + 1].toFixed(6)},${tilePositions[t * 3 + 2].toFixed(6)}`
    const v = vertexAt.get(key)
    if (v === undefined) throw new Error(`tile ${t + 1} is not a vertex of the detail-${DETAIL} icosahedron`)
    vertexOfTile[t] = v
}
say(`${coordinatesName}: ${tiles} tiles, all on the lattice`)

const bordersName = only("static", /^borders-[0-9a-f]{8}\.bin$/)
const borders = fs.readFileSync(`static/${bordersName}`)
const headerBytes = new DataView(borders.buffer, borders.byteOffset, borders.byteLength).getUint32(0, true)
const header = JSON.parse(new TextDecoder().decode(new Uint8Array(borders.buffer, borders.byteOffset + 4, headerBytes)))
if (header.tiles !== tiles) throw new Error(`${bordersName} covers ${header.tiles} tiles, ${coordinatesName} has ${tiles}`)
const assignment = new Uint16Array(borders.buffer, borders.byteOffset + 4 + headerBytes, tiles)

const countryOfLandmass = new Uint16Array(header.codes.length)
const countryCount = new Map([["", 0]])
for (let piece = 1; piece < header.codes.length; piece++) {
    const code = header.codes[piece]
    let index = countryCount.get(code)
    if (index === undefined) countryCount.set(code, index = countryCount.size)
    countryOfLandmass[piece] = index
}
say(`${bordersName}: ${countryCount.size - 1} countries across ${header.codes.length - 1} landmasses`)

const country = new Uint16Array(vertices)
for (let t = 0; t < tiles; t++) country[vertexOfTile[t]] = countryOfLandmass[assignment[t]]

const centres = new Float64Array(faces * 3)
const haveCentre = new Uint8Array(faces)
function centreOf(face) {
    if (!haveCentre[face]) {
        const [a, b, c] = [corners[face * 3], corners[face * 3 + 1], corners[face * 3 + 2]]
        const ux = px[b] - px[a], uy = py[b] - py[a], uz = pz[b] - pz[a]
        const wx = px[c] - px[a], wy = py[c] - py[a], wz = pz[c] - pz[a]
        let nx = uy * wz - uz * wy, ny = uz * wx - ux * wz, nz = ux * wy - uy * wx
        const length = Math.hypot(nx, ny, nz) || 1
        const sign = (nx * px[a] + ny * py[a] + nz * pz[a]) < 0 ? -1 : 1
        centres[face * 3] = sign * nx / length
        centres[face * 3 + 1] = sign * ny / length
        centres[face * 3 + 2] = sign * nz / length
        haveCentre[face] = 1
    }
    return face
}

const pending = new Map()
const segments = []
let land = 0
for (let f = 0; f < faces; f++) {
    for (let e = 0; e < 3; e++) {
        const a = corners[f * 3 + e], b = corners[f * 3 + (e + 1) % 3]
        const [first, second] = a < b ? [a, b] : [b, a]
        if (country[first] === country[second]) continue
        const key = first * vertices + second
        const other = pending.get(key)
        if (other === undefined) { pending.set(key, f); continue }
        pending.delete(key)
        segments.push([centreOf(other), centreOf(f)])
        if (country[first] && country[second]) land++
    }
}
if (pending.size) throw new Error(`${pending.size} cell edges have only one triangle`)
say(`outline: ${segments.length} cell edges (${land} between two countries, ${segments.length - land} coast)`)

const onCorner = new Map()
for (let s = 0; s < segments.length; s++) {
    for (const corner of segments[s]) {
        let list = onCorner.get(corner)
        if (!list) onCorner.set(corner, list = [])
        list.push(s)
    }
}
const junction = (corner) => onCorner.get(corner).length !== 2
const walked = new Uint8Array(segments.length)
const runs = []
const walk = (from) => {
    const run = [from]
    let at = from
    for (;;) {
        let next = -1
        for (const s of onCorner.get(at)) if (!walked[s]) { next = s; break }
        if (next < 0) return run
        walked[next] = 1
        at = segments[next][0] === at ? segments[next][1] : segments[next][0]
        run.push(at)
        if (junction(at)) return run
    }
}
const take = (run) => {
    if (run.length > 2 && run[0] === run[run.length - 1] && junction(run[0])) {
        const half = run.length >> 1
        runs.push(run.slice(0, half + 1), run.slice(half))
        return
    }
    runs.push(run)
}
for (const [corner, list] of onCorner) if (junction(corner)) while (list.some((s) => !walked[s])) take(walk(corner))
for (const [corner, list] of onCorner) while (list.some((s) => !walked[s])) take(walk(corner))
const closed = runs.filter((run) => run[0] === run[run.length - 1]).length
const written = runs.reduce((n, run) => n + run.length, 0)
say(`chained into ${runs.length} runs (${closed} closed) past ${[...onCorner.keys()].filter(junction).length} junctions,`
    + ` ${written} corners written for ${segments.length} edges`)

const points = new Int16Array(written * 3)
let at = 0
for (const run of runs) {
    for (const corner of run) {
        points[at++] = Math.round(centres[corner * 3] * 32767)
        points[at++] = Math.round(centres[corner * 3 + 1] * 32767)
        points[at++] = Math.round(centres[corner * 3 + 2] * 32767)
    }
}
const lengths = new Uint32Array(runs.length)
for (let r = 0; r < runs.length; r++) lengths[r] = runs[r].length

// A 16-byte head keeps lengths 4-byte and corners 2-byte aligned, for zero-copy views.
const head = Buffer.alloc(16)
head.write("CPBL", 0, "ascii")
head.writeUInt32LE(1, 4)
head.writeUInt32LE(runs.length, 8)
head.writeUInt32LE(written, 12)
const bytes = Buffer.concat([head, Buffer.from(lengths.buffer), Buffer.from(points.buffer)])

const hash = createHash("sha256").update(bytes).digest("hex").slice(0, 8)
const fileName = `borderLines-${hash}.bin`
for (const entry of fs.readdirSync("static")) {
    if (/^borderLines-[0-9a-f]{8}\.bin$/.test(entry) && entry !== fileName) fs.unlinkSync(`static/${entry}`)
}
fs.writeFileSync(`static/${fileName}`, bytes)
fs.writeFileSync("src/app/viewer/borderLinesAsset.ts",
    `export const BORDER_LINES_URL = "/static/${fileName}"
`)
say(`wrote static/${fileName}: ${bytes.byteLength} bytes`)

function only(directory, pattern) {
    const found = fs.readdirSync(directory).filter((entry) => pattern.test(entry))
    if (found.length !== 1) throw new Error(`expected one ${directory}/${pattern}, found ${found.length}`)
    return found[0]
}
