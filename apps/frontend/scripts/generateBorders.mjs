// Resolves every tile to the piece of land it sits on, and writes the table the
// zoomed-out view paints flags from.
//
// Borders come from Natural Earth 1:50m — public domain, one file, ISO codes in
// the properties. OpenStreetMap has the same borders as admin_level=2 relations,
// but you would have to run an Overpass query and stitch ways into rings
// yourself, and it is ODbL, so share-alike and attribution come with it.
//
//   node scripts/generateBorders.mjs [path-to-geojson] [out]
//
// With no geojson path it downloads the dataset. Nothing about the output
// changes unless the dataset or the coordinates blob does, so this is run by
// hand, not on every build.
import fs from "node:fs"

const SOURCE = "https://raw.githubusercontent.com/nvkelso/natural-earth-vector/master/geojson/ne_50m_admin_0_countries.geojson"

const given = process.argv[2] && !process.argv[2].startsWith("-") ? process.argv[2] : undefined
const geo = given
    ? JSON.parse(fs.readFileSync(given, "utf8"))
    : await (await fetch(SOURCE)).json()

// --- tiles
const buf = fs.readFileSync("static/coordinates-26a9aeab.bin")
const dv = new DataView(buf.buffer, buf.byteOffset, buf.byteLength)
const N = dv.getUint32(8, true)
const pos = new Float32Array(buf.buffer, buf.byteOffset + 12, N * 3)
const uvs = new Float32Array(buf.buffer, buf.byteOffset + 12 + N * 12, N * 2)

// --- polygons, flattened to rings with bboxes
const shapes = []   // {code, rings: [[x,y,...]], bbox}
for (const f of geo.features) {
    const code = (f.properties.ISO_A2_EH ?? f.properties.ISO_A2 ?? "").toLowerCase()
    if (!code || code === "-99") continue
    const polys = f.geometry.type === "Polygon" ? [f.geometry.coordinates] : f.geometry.coordinates
    for (const poly of polys) {
        let minX = 180, minY = 90, maxX = -180, maxY = -90
        const rings = poly.map((ring) => {
            const flat = new Float64Array(ring.length * 2)
            for (let i = 0; i < ring.length; i++) {
                const [x, y] = ring[i]
                flat[i * 2] = x; flat[i * 2 + 1] = y
                if (x < minX) minX = x
                if (x > maxX) maxX = x
                if (y < minY) minY = y
                if (y > maxY) maxY = y
            }
            return flat
        })
        shapes.push({code, rings, bbox: [minX, minY, maxX, maxY]})
    }
}
console.error(`shapes: ${shapes.length}`)

// --- 1-degree grid over the shapes' bboxes
const GX = 360, GY = 180
const grid = new Map()
const key = (i, j) => j * GX + i
for (let s = 0; s < shapes.length; s++) {
    const [minX, minY, maxX, maxY] = shapes[s].bbox
    for (let j = Math.floor(minY + 90); j <= Math.min(GY - 1, Math.floor(maxY + 90)); j++) {
        for (let i = Math.floor(minX + 180); i <= Math.min(GX - 1, Math.floor(maxX + 180)); i++) {
            const k = key(Math.max(0, i), Math.max(0, j))
            let arr = grid.get(k); if (!arr) grid.set(k, arr = []); arr.push(s)
        }
    }
}

function inRing(ring, x, y) {
    let inside = false
    const n = ring.length / 2
    for (let i = 0, j = n - 1; i < n; j = i++) {
        const xi = ring[i * 2], yi = ring[i * 2 + 1]
        const xj = ring[j * 2], yj = ring[j * 2 + 1]
        if ((yi > y) !== (yj > y) && x < (xj - xi) * (y - yi) / (yj - yi) + xi) inside = !inside
    }
    return inside
}

function shapeAt(x, y) {
    const i = Math.min(GX - 1, Math.max(0, Math.floor(x + 180)))
    const j = Math.min(GY - 1, Math.max(0, Math.floor(y + 90)))
    const candidates = grid.get(key(i, j))
    if (!candidates) return -1
    for (const s of candidates) {
        const {rings, bbox} = shapes[s]
        if (x < bbox[0] || x > bbox[2] || y < bbox[1] || y > bbox[3]) continue
        if (!inRing(rings[0], x, y)) continue
        let hole = false
        for (let r = 1; r < rings.length && !hole; r++) hole = inRing(rings[r], x, y)
        if (!hole) return s
    }
    return -1
}

// --- two candidate lon/lat conventions; pick whichever lands more tiles on land
const fromUV = (t) => [(uvs[t * 2] - 0.5) * 360, (uvs[t * 2 + 1] - 0.5) * 180]
const fromXYZ = (t) => {
    const [x, y, z] = [pos[t * 3], pos[t * 3 + 1], pos[t * 3 + 2]]
    return [Math.atan2(z, -x) * 180 / Math.PI, Math.asin(Math.max(-1, Math.min(1, y))) * 180 / Math.PI]
}
const sample = []
for (let i = 0; i < 4000; i++) sample.push((Math.random() * N) | 0)
for (const [name, fn] of [["uv", fromUV], ["xyz", fromXYZ]]) {
    let hits = 0
    for (const t of sample) if (shapeAt(...fn(t)) !== -1) hits++
    console.error(`${name}: ${(hits / sample.length * 100).toFixed(1)}% of sampled tiles land inside a country`)
}

const out = process.argv[3] ?? "public/dev-borders.bin"
const project = fromUV
const country = new Int32Array(N).fill(-1)
const counts = new Map()
for (let t = 0; t < N; t++) {
    const s = shapeAt(...project(t))
    if (s === -1) continue
    country[t] = s
    const code = shapes[s].code
    counts.set(code, (counts.get(code) ?? 0) + 1)
}

const assigned = [...counts.values()].reduce((a, b) => a + b, 0)
console.error(`assigned ${assigned}/${N} tiles to ${counts.size} countries`)

// --- split each country into the separate landmasses it is made of.
//
// A flag belongs to a piece of land, not to a sovereignty. France is mainland
// *and* New Caledonia; one frame spanning both would stretch the tricolour
// across half the planet and paint nothing recognisable in either place. Two
// tiles are the same piece when they are neighbours on the tile lattice and
// carry the same country.
const NEIGHBOUR = 0.00534   // 1.35x the median spacing between tiles

const BL = 400, BLON = 800
const tileBins = new Map()
const tileKey = (i, j) => i * BLON + ((j % BLON) + BLON) % BLON
const lat = new Float64Array(N), lon = new Float64Array(N)
for (let t = 0; t < N; t++) {
    lat[t] = Math.asin(Math.max(-1, Math.min(1, pos[t * 3 + 1])))
    lon[t] = Math.atan2(pos[t * 3 + 2], pos[t * 3])
    const i = Math.min(BL - 1, Math.floor((lat[t] + Math.PI / 2) / Math.PI * BL))
    const j = Math.floor((lon[t] + Math.PI) / (2 * Math.PI) * BLON)
    const k = tileKey(i, j)
    let arr = tileBins.get(k); if (!arr) tileBins.set(k, arr = []); arr.push(t)
}

const parent = new Int32Array(N)
for (let t = 0; t < N; t++) parent[t] = t
const find = (x) => { while (parent[x] !== x) x = parent[x] = parent[parent[x]]; return x }

const spanLat = Math.ceil(NEIGHBOUR / (Math.PI / BL))
for (let t = 0; t < N; t++) {
    if (country[t] === -1) continue
    const code = shapes[country[t]].code
    const i = Math.min(BL - 1, Math.floor((lat[t] + Math.PI / 2) / Math.PI * BL))
    const j = Math.floor((lon[t] + Math.PI) / (2 * Math.PI) * BLON)
    const spanLon = Math.ceil(NEIGHBOUR / (2 * Math.PI / BLON) / Math.max(0.05, Math.cos(lat[t])))
    for (let di = -spanLat; di <= spanLat; di++) {
        const ii = i + di; if (ii < 0 || ii >= BL) continue
        for (let dj = -spanLon; dj <= spanLon; dj++) {
            const arr = tileBins.get(tileKey(ii, j + dj)); if (!arr) continue
            for (const u of arr) {
                if (u <= t || country[u] === -1) continue
                if (shapes[country[u]].code !== code) continue
                const dx = pos[t * 3] - pos[u * 3]
                const dy = pos[t * 3 + 1] - pos[u * 3 + 1]
                const dz = pos[t * 3 + 2] - pos[u * 3 + 2]
                if (dx * dx + dy * dy + dz * dz > NEIGHBOUR * NEIGHBOUR) continue
                const a = find(t), b = find(u)
                if (a !== b) parent[a] = b
            }
        }
    }
}

const codes = [""]
const slotOf = new Map()
const assignment = new Uint16Array(N)
for (let t = 0; t < N; t++) {
    if (country[t] === -1) continue
    const root = find(t)
    let k = slotOf.get(root)
    if (k === undefined) {
        k = codes.length
        if (k > 65535) throw new Error("more landmasses than a Uint16 can name")
        codes.push(shapes[country[t]].code)
        slotOf.set(root, k)
    }
    assignment[t] = k
}

const pieceSize = new Map()
for (let t = 0; t < N; t++) if (assignment[t]) pieceSize.set(assignment[t], (pieceSize.get(assignment[t]) ?? 0) + 1)
const perCountry = new Map()
for (const [k, n] of pieceSize) {
    const c = codes[k]
    let e = perCountry.get(c); if (!e) perCountry.set(c, e = [])
    e.push(n)
}
console.error(`${codes.length - 1} landmasses across ${perCountry.size} countries`)
console.error("most fragmented: " + [...perCountry].sort((a, b) => b[1].length - a[1].length).slice(0, 8)
    .map(([c, list]) => `${c}:${list.length}`).join(" "))
for (const c of ["fr", "us", "gb", "es", "nl", "dk"]) {
    const list = (perCountry.get(c) ?? []).sort((a, b) => b - a)
    console.error(`  ${c}: ${list.length} pieces, biggest ${list.slice(0, 5).join(",")}`)
}

// --- a frame per country, for painting a flag across it. Borders do not move,
// so this is static: centre direction, the east axis there, and how far the
// country's tiles reach along each axis measured over the surface.
const frames = new Float32Array(codes.length * 5)
const members = Array.from({length: codes.length}, () => [])
for (let t = 0; t < N; t++) if (assignment[t]) members[assignment[t]].push(t)

for (let k = 1; k < codes.length; k++) {
    const own = members[k]
    if (own.length === 0) continue

    let [mx, my, mz] = [0, 0, 0]
    for (const t of own) { mx += pos[t * 3]; my += pos[t * 3 + 1]; mz += pos[t * 3 + 2] }
    let len2 = Math.hypot(mx, my, mz) || 1
    let centre = [mx / len2, my / len2, mz / len2]

    // The mean direction of a country curved over a sphere can fall outside it,
    // so snap to the country's own tile nearest that direction.
    let best = own[0], bestDot = -2
    for (const t of own) {
        const d = pos[t * 3] * centre[0] + pos[t * 3 + 1] * centre[1] + pos[t * 3 + 2] * centre[2]
        if (d > bestDot) { bestDot = d; best = t }
    }
    centre = [pos[best * 3], pos[best * 3 + 1], pos[best * 3 + 2]]
    len2 = Math.hypot(...centre) || 1
    centre = centre.map((v) => v / len2)

    const flat = Math.hypot(centre[0], centre[2])
    const east = flat < 1e-4 ? [1, 0, 0] : [centre[2] / flat, 0, -centre[0] / flat]
    const north = [
        centre[1] * east[2] - centre[2] * east[1],
        centre[2] * east[0] - centre[0] * east[2],
        centre[0] * east[1] - centre[1] * east[0],
    ]

    let halfU = 0, halfV = 0
    for (const t of own) {
        const p = [pos[t * 3], pos[t * 3 + 1], pos[t * 3 + 2]]
        const along = p[0] * centre[0] + p[1] * centre[1] + p[2] * centre[2]
        if (along <= 0) continue
        const angle = Math.acos(Math.min(1, along))
        const u = p[0] * east[0] + p[1] * east[1] + p[2] * east[2]
        const v = p[0] * north[0] + p[1] * north[1] + p[2] * north[2]
        const l = Math.hypot(u, v) || 1
        halfU = Math.max(halfU, Math.abs(angle * u / l))
        halfV = Math.max(halfV, Math.abs(angle * v / l))
    }

    frames[k * 5] = centre[0]
    frames[k * 5 + 1] = centre[1]
    frames[k * 5 + 2] = centre[2]
    frames[k * 5 + 3] = halfU
    frames[k * 5 + 4] = halfV
}

const totals = new Uint32Array(codes.length)
for (let k = 1; k < codes.length; k++) totals[k] = members[k].length

// Padded so the float sections that follow stay 4-byte aligned and the decoder
// can take zero-copy views.
let json = JSON.stringify({tiles: N, codes})
while ((4 + Buffer.byteLength(json, "utf8")) % 4 !== 0) json += " "
const header = Buffer.from(json, "utf8")
const len = Buffer.alloc(4); len.writeUInt32LE(header.length, 0)
fs.writeFileSync(out, Buffer.concat([
    len, header,
    Buffer.from(assignment.buffer),
    Buffer.from(frames.buffer),
    Buffer.from(totals.buffer),
]))
console.error(`wrote ${out}: ${codes.length - 1} landmasses`)
