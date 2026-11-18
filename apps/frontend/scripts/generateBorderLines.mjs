import {encodeBorderLines} from "../src/domain/borderLinesFile.ts"
import {gridDirection, latticeSize, latticeTriangles, triangleOnGrid} from "../src/domain/lattice.ts"
import {encodeProvinceLines, fileHashOf} from "../src/domain/provinceLinesFile.ts"
import {boardDir, readBoard, readBorders, readProvinces} from "./map/board.mjs"
import {boardsAsked, writeBoardsTable, writePageFile} from "./writeBoard.ts"

const say = (...args) => console.error(...args)

// `npm run borderLines -- earth-1` traces one board, and with no name every board in /map.
for (const name of boardsAsked(process.argv.slice(2))) traceBorderLines(name)
writeBoardsTable()

function traceBorderLines(name) {
    const board = readBoard(boardDir(name))
    const {lattice: shape} = board.manifest
    say(`walking the lattice of ${board.manifest.name} at cols ${shape.cols}...`)
    const corners = latticeTriangles(shape)
    const vertices = latticeSize(shape.cols)
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

    // A cell corner is where the ground's cells meet (latticeCells.ts): the territory layer and the pick find a direction's
    // cell on its face's grid, so three cells meet at each triangle's centroid on that grid, warped and projected as a
    // lattice point is. The outline is drawn on the cells the ground paints, not on another tiling of the same tiles.
    const centres = new Float64Array(faces * 3)
    const haveCentre = new Uint8Array(faces)
    const grid = new Int32Array(7)
    function centreOf(face) {
        if (!haveCentre[face]) {
            triangleOnGrid(shape.cols, face, grid)
            gridDirection(shape, grid[0], (grid[1] + grid[3] + grid[5]) / 3, (grid[2] + grid[4] + grid[6]) / 3, centres, face * 3)
            haveCentre[face] = 1
        }
        return face
    }

    // A cell edge on an edge of the icosahedron runs from one face's grid into the next, and bends where it crosses: at
    // the middle of its lattice edge on both grids. One corner past the triangles' for each, the same for every kind.
    const middles = new Map()
    const middleCorners = []
    function middleOf(triangle, side, key) {
        let corner = middles.get(key)
        if (corner === undefined) {
            triangleOnGrid(shape.cols, triangle, grid)
            const next = (side + 1) % 3
            const at = new Float64Array(3)
            gridDirection(shape, grid[0], (grid[1 + 2 * side] + grid[1 + 2 * next]) / 2, (grid[2 + 2 * side] + grid[2 + 2 * next]) / 2, at)
            corner = faces + middleCorners.length
            middleCorners.push(at)
            middles.set(key, corner)
        }
        return corner
    }
    const faceOf = (triangle) => Math.floor(triangle / (shape.cols * shape.cols))
    const cornerAt = (corner) => corner < faces ? centres.subarray(corner * 3, corner * 3 + 3) : middleCorners[corner - faces]

    // Every cell edge between two lattice points whose `side` differs, once (in two pieces where it bends from one face
    // into the next), with the two points either side of each piece.
    const edgesBetween = (side) => {
        const pending = new Map()
        const segments = []
        const ends = []
        for (let f = 0; f < faces; f++) {
            for (let e = 0; e < 3; e++) {
                const a = corners[f * 3 + e], b = corners[f * 3 + (e + 1) % 3]
                const [first, second] = a < b ? [a, b] : [b, a]
                if (!side(first, second)) continue
                const key = first * vertices + second
                const other = pending.get(key)
                if (other === undefined) { pending.set(key, f); continue }
                pending.delete(key)
                if (faceOf(other) === faceOf(f)) {
                    segments.push([centreOf(other), centreOf(f)])
                    ends.push([first, second])
                    continue
                }
                const middle = middleOf(f, e, key)
                segments.push([centreOf(other), middle], [middle, centreOf(f)])
                ends.push([first, second], [first, second])
            }
        }
        if (pending.size) throw new Error(`${pending.size} cell edges have only one triangle`)
        return {segments, ends}
    }

    const outline = edgesBetween((a, b) => country[a] !== country[b])
    const land = outline.ends.filter(([a, b]) => country[a] && country[b]).length
    say(`outline: ${outline.segments.length} pieces (${land} between two countries, ${outline.segments.length - land} coast)`)
    const sideOf = (values) => ([a, b]) => Math.min(values[a], values[b]) * 65536 + Math.max(values[a], values[b])
    const {runs, runSides, points, lengths} = chained(outline.segments, outline.ends.map(sideOf(landmass)), "landmasses")

    const bytes = Buffer.from(encodeBorderLines({runs: lengths, sides: runSides, corners: points}))

    const fileName = writePageFile(name, "borderLines", bytes)
    say(`wrote static/boards/${name}/${fileName}: ${bytes.byteLength} bytes`)
    if (board.manifest.files.provinces) traceProvinceLines(fileName, runs, outline, landmass)

    // Corners in turn, each run's ending at a junction; a loop with none is cut in two at its start.
    function chained(segments, sides, what) {
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

        // Two tiles of one country that touch are one landmass, and two of one province one province: a run between two
        // junctions never changes either side.
        const runSides = new Uint16Array(runs.length * 2)
        runs.forEach(({edges}, r) => {
            const pair = sides[edges[0]]
            if (edges.some((edge) => sides[edge] !== pair)) throw new Error(`run ${r} borders more than two ${what}`)
            runSides[r * 2] = Math.floor(pair / 65536)
            runSides[r * 2 + 1] = pair % 65536
        })
        say(`chained into ${runs.length} runs (${closed} closed) past ${[...onCorner.keys()].filter(junction).length} junctions,`
            + ` ${written} corners written for ${segments.length} edges`)

        const points = new Int16Array(written * 3)
        let at = 0
        for (const {corners} of runs) {
            for (const corner of corners) {
                const [x, y, z] = cornerAt(corner)
                points[at++] = Math.round(x * 32767)
                points[at++] = Math.round(y * 32767)
                points[at++] = Math.round(z * 32767)
            }
        }
        const lengths = new Uint32Array(runs.length)
        for (let r = 0; r < runs.length; r++) lengths[r] = runs[r].corners.length
        return {runs, runSides, points, lengths}
    }

    // The cell edges between two provinces of one country, and the provinces either side of each piece of the countries'.
    function traceProvinceLines(countryLines, countryRuns, outline, landmass) {
        const {name: file, provinces} = readProvinces(board)
        const province = new Uint16Array(vertices)
        for (let t = 0; t < tiles; t++) province[vertexOfTile[t]] = provinces.province[t]
        say(`${file}: ${provinces.landmass.length - 1} provinces`)

        const inner = edgesBetween((a, b) => country[a] === country[b] && country[a] !== 0 && province[a] !== province[b])
        say(`province lines: ${inner.segments.length} pieces`)
        const lines = chained(inner.segments, inner.ends.map(sideOf(province)), "provinces")

        // A piece's first province is on its run's first landmass's side: the lesser landmass, as the run's pair is.
        const countrySides = new Uint16Array(outline.segments.length * 2)
        let piece = 0
        for (const {edges} of countryRuns) {
            for (const edge of edges) {
                const [a, b] = outline.ends[edge]
                const [lesser, greater] = landmass[a] < landmass[b] ? [a, b] : [b, a]
                countrySides[piece * 2] = province[lesser]
                countrySides[piece * 2 + 1] = province[greater]
                piece++
            }
        }

        const bytes = Buffer.from(encodeProvinceLines({
            countryLines: fileHashOf(countryLines),
            provinces: fileHashOf(file),
            lines: {runs: lines.lengths, sides: lines.runSides, corners: lines.points},
            countrySides,
        }))
        const fileName = writePageFile(name, "provinceLines", bytes)
        say(`wrote static/boards/${name}/${fileName}: ${bytes.byteLength} bytes`)
    }
}
