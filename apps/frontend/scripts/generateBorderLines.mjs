import {encodeBorderLines} from "../src/domain/borderLinesFile.ts"
import {latticePositions, latticeSize, latticeTriangles} from "../src/domain/lattice.ts"
import {boardDir, readBoard, readBorders} from "./map/board.mjs"
import {boardsAsked, writeBoardsTable, writePageFile} from "./writeBoard.ts"

const say = (...args) => console.error(...args)

// `npm run borderLines -- earth-1` traces one board, and with no name every board in /map.
for (const name of boardsAsked(process.argv.slice(2))) traceBorderLines(name)
writeBoardsTable()

function traceBorderLines(name) {
    const board = readBoard(boardDir(name))
    const {lattice: shape} = board.manifest
    say(`walking the lattice of ${board.manifest.name} at cols ${shape.cols}...`)
    const lattice = latticePositions(shape)
    const px = [], py = [], pz = []
    for (let v = 0; v < latticeSize(shape.cols); v++) {
        px.push(lattice[v * 3]); py.push(lattice[v * 3 + 1]); pz.push(lattice[v * 3 + 2])
    }
    const corners = latticeTriangles(shape)
    const vertices = px.length
    const faces = corners.length / 3
    say(`lattice: ${vertices} vertices, ${faces} triangles`)

    const tiles = board.count
    const vertexOfTile = board.points
    say(`${board.file}: ${tiles} tiles`)

    const borders = readBorders(board)
    const assignment = borders.landmass

    const countryOfLandmass = new Uint16Array(borders.codes.length)
    const countryCount = new Map([["", 0]])
    for (let piece = 1; piece < borders.codes.length; piece++) {
        const code = borders.codes[piece]
        let index = countryCount.get(code)
        if (index === undefined) countryCount.set(code, index = countryCount.size)
        countryOfLandmass[piece] = index
    }
    say(`${borders.name}: ${countryCount.size - 1} countries across ${borders.codes.length - 1} landmasses`)

    const country = new Uint16Array(vertices)
    const landmass = new Uint16Array(vertices)
    for (let t = 0; t < tiles; t++) {
        country[vertexOfTile[t]] = countryOfLandmass[assignment[t]]
        landmass[vertexOfTile[t]] = assignment[t]
    }

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
    const sides = []
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
            sides.push(Math.min(landmass[first], landmass[second]) * 65536 + Math.max(landmass[first], landmass[second]))
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
        const run = {corners: [from], edges: []}
        let at = from
        for (;;) {
            let next = -1
            for (const s of onCorner.get(at)) if (!walked[s]) { next = s; break }
            if (next < 0) return run
            walked[next] = 1
            run.edges.push(next)
            at = segments[next][0] === at ? segments[next][1] : segments[next][0]
            run.corners.push(at)
            if (junction(at)) return run
        }
    }
    const take = ({corners, edges}) => {
        if (corners.length > 2 && corners[0] === corners[corners.length - 1] && junction(corners[0])) {
            const half = corners.length >> 1
            runs.push({corners: corners.slice(0, half + 1), edges: edges.slice(0, half)}, {corners: corners.slice(half), edges: edges.slice(half)})
            return
        }
        runs.push({corners, edges})
    }
    for (const [corner, list] of onCorner) if (junction(corner)) while (list.some((s) => !walked[s])) take(walk(corner))
    for (const [corner, list] of onCorner) while (list.some((s) => !walked[s])) take(walk(corner))
    const closed = runs.filter(({corners}) => corners[0] === corners[corners.length - 1]).length
    const written = runs.reduce((n, {corners}) => n + corners.length, 0)

    // Two tiles of one country that touch are one landmass, so a run never changes landmass between two junctions.
    const runSides = new Uint16Array(runs.length * 2)
    runs.forEach(({edges}, r) => {
        const pair = sides[edges[0]]
        if (edges.some((edge) => sides[edge] !== pair)) throw new Error(`run ${r} borders more than two landmasses`)
        runSides[r * 2] = Math.floor(pair / 65536)
        runSides[r * 2 + 1] = pair % 65536
    })
    say(`chained into ${runs.length} runs (${closed} closed) past ${[...onCorner.keys()].filter(junction).length} junctions,`
        + ` ${written} corners written for ${segments.length} edges`)

    const points = new Int16Array(written * 3)
    let at = 0
    for (const {corners} of runs) {
        for (const corner of corners) {
            points[at++] = Math.round(centres[corner * 3] * 32767)
            points[at++] = Math.round(centres[corner * 3 + 1] * 32767)
            points[at++] = Math.round(centres[corner * 3 + 2] * 32767)
        }
    }
    const lengths = new Uint32Array(runs.length)
    for (let r = 0; r < runs.length; r++) lengths[r] = runs[r].corners.length

    const bytes = Buffer.from(encodeBorderLines({runs: lengths, sides: runSides, corners: points}))

    const fileName = writePageFile(name, "borderLines", bytes)
    say(`wrote static/boards/${name}/${fileName}: ${bytes.byteLength} bytes`)
}
