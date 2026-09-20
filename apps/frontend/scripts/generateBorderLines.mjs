// Draws the countries' outlines onto the tile lattice, and writes the blob the
// globe renders them from.
//
//   npm run borderLines
//
// The line is not Natural Earth's polygon. It is the boundary of each country's
// *tiles*: every tile is the centre of a hexagonal cell (the tiles are the
// vertices of a geodesic sphere, so the cells are its dual, a Goldberg solid),
// and the outline runs along the cell edges between two tiles that belong to
// different countries. So the administrative border is moved, by up to half a
// tile — about 12 km — onto the lattice.
//
// That is the whole point of doing it this way. A line drawn on the real border
// crosses tiles, and a crossed tile belongs to one side while reading as split
// between both. A line on the cell edges passes *between* the discs at every
// zoom — the tile field is 76% covered once zoomed in, and the gap it leaves is
// exactly where this runs — so a tile is never cut and which side it is on is
// never in doubt.
//
// Both inputs are static and neither moves: the coordinates blob fixes where
// the tiles are, the borders blob fixes which country each sits in. So this is
// run by hand when one of them changes, not on every build, and it writes
// static/borderLines-<hash>.bin and the module naming it — the same
// content-addressed pair as the coordinates blob, the borders blob and the
// atlas, because `public/_headers` caches /static/* for a week and a
// regenerated file under a stable name would be served stale.
//
// It stays out of /map, unlike the two blobs it is built from: the backend has
// no use for it. Where a tile is and who owns the ground under it are the game's
// rules and are shared; how thick a line is drawn between them is this app's.
import fs from "node:fs"
import {createHash} from "node:crypto"
import * as THREE from "three"

// The detail the coordinates blob was generated at. Not recorded in it — see
// map/README.md, which recovers it and explains how it was checked. The
// reconstruction below asserts it: every tile has to land on a lattice vertex.
const DETAIL = 300

// A country-less vertex takes its neighbours' country when at least this many
// of the six agree and none of them disagrees. It closes the pinholes the land
// mask left — a one-tile lake, a strait one tile wide, the thin row the
// antimeridian drops — which would otherwise each get an outline of their own
// and fray every coast. Below 4 it starts eating into real bays.
const HEAL_NEIGHBOURS = 4

// Twice, so a two-vertex pinhole closes as well. A third pass starts rounding
// off real inlets.
const HEAL_PASSES = 2

const say = (...args) => console.error(...args)

// --- the lattice the tiles were cut from.
//
// The coordinates blob holds only the land vertices, so the sea around a
// coast — and the country a tile borders — is not in it. Rebuilding the whole
// icosahedron is what gives every tile its six neighbours *and* the cell
// corners the outline is drawn through, exactly rather than by searching.
say(`rebuilding the icosahedron at detail ${DETAIL}...`)
const raw = new THREE.IcosahedronGeometry(1, DETAIL).attributes.position.array

// The same key computeCoordinates.ts deduplicated with, so a tile's position
// finds the vertex it was taken from.
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

// --- the tiles, and which lattice vertex each one is.
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

// --- which country each tile sits in. The borders blob names a landmass per
// tile; two landmasses of the same country are by construction never
// neighbours, so the outline is drawn on the country, not on the piece.
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

// 0 is "no country": the sea, and the tiles that fall outside every polygon.
const country = new Uint16Array(vertices)
for (let t = 0; t < tiles; t++) country[vertexOfTile[t]] = countryOfLandmass[assignment[t]]

// --- who neighbours whom. Every interior vertex has exactly six neighbours and
// the twelve icosahedron corners have five, so the lists are a fixed stride.
const degree = new Uint8Array(vertices)
const neighbours = new Int32Array(vertices * 6)
const link = (a, b) => {
    for (let i = 0; i < degree[a]; i++) if (neighbours[a * 6 + i] === b) return
    neighbours[a * 6 + degree[a]++] = b
}
for (let f = 0; f < corners.length; f += 3) {
    const [a, b, c] = [corners[f], corners[f + 1], corners[f + 2]]
    link(a, b); link(b, a); link(b, c); link(c, b); link(c, a); link(a, c)
}

// --- close the pinholes, so a coast is a line and not a dotted one.
for (let pass = 0; pass < HEAL_PASSES; pass++) {
    const next = Uint16Array.from(country)
    let filled = 0
    for (let v = 0; v < vertices; v++) {
        if (country[v]) continue
        let around = 0, one = 0, mixed = false
        for (let i = 0; i < degree[v]; i++) {
            const c = country[neighbours[v * 6 + i]]
            if (!c) continue
            around++
            if (!one) one = c
            else if (one !== c) mixed = true
        }
        if (around >= HEAL_NEIGHBOURS && !mixed) { next[v] = one; filled++ }
    }
    country.set(next)
    say(`heal pass ${pass + 1}: ${filled} vertices took a neighbouring country`)
}

// --- the outline itself.
//
// A cell edge is the dual of a lattice edge: it runs between the circumcentres
// of the two triangles that share it. The circumcentre of a spherical triangle
// is the normal of the plane through its three vertices, which is what makes
// the cells a true Voronoi diagram of the tiles and the corners meet exactly —
// no seams to cover up at the joins.
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

// An edge is drawn when its two ends are in different countries and at least
// one of them is in a country at all — a stretch of sea, or the tiles that fall
// outside every polygon, is nobody's ground and gets no outline.
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

// --- chain the edges into runs, so a corner is written once instead of twice.
// Each run is a polyline; the renderer draws a quad per edge either way, this
// is only what the file costs. Three countries meeting is a corner with three
// edges on it, so the runs are not all loops and a walk has to start at the
// odd ends before it starts anywhere.
const onCorner = new Map()
for (let s = 0; s < segments.length; s++) {
    for (const corner of segments[s]) {
        let list = onCorner.get(corner)
        if (!list) onCorner.set(corner, list = [])
        list.push(s)
    }
}
const walked = new Uint8Array(segments.length)
const runs = []
const walk = (from) => {
    const run = [from]
    let at = from
    for (;;) {
        const list = onCorner.get(at)
        let next = -1
        for (const s of list) if (!walked[s]) { next = s; break }
        if (next < 0) return run
        walked[next] = 1
        at = segments[next][0] === at ? segments[next][1] : segments[next][0]
        run.push(at)
    }
}
for (const [corner, list] of onCorner) if (list.length % 2 === 1) while (list.some((s) => !walked[s])) runs.push(walk(corner))
for (const [corner, list] of onCorner) while (list.some((s) => !walked[s])) runs.push(walk(corner))
const written = runs.reduce((n, run) => n + run.length, 0)
say(`chained into ${runs.length} runs, ${written} corners written for ${segments.length} edges`)

// --- write it. The corners are on the unit sphere and the renderer normalizes
// what it reads, so a signed 16-bit share of the radius is plenty: the worst
// error is 3e-5 of the radius, which is a two-hundredth of the gap between two
// tiles.
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

// "CPBL" | uint32 version | uint32 run count | uint32 corner count | run lengths | corners.
// The 16-byte head keeps the lengths 4-byte aligned and the corners 2-byte
// aligned, so the browser takes zero-copy views of both.
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
    `// Generated by scripts/generateBorderLines.mjs — do not edit by hand.
// Regenerate with \`npm run borderLines\`.

export const BORDER_LINES_URL = "/static/${fileName}"
`)
say(`wrote static/${fileName}: ${bytes.byteLength} bytes`)

function only(directory, pattern) {
    const found = fs.readdirSync(directory).filter((entry) => pattern.test(entry))
    if (found.length !== 1) throw new Error(`expected one ${directory}/${pattern}, found ${found.length}`)
    return found[0]
}
