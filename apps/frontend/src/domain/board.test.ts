import {readdirSync, readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import {BOARD} from "../app/viewer/boardAsset.ts"
import {decodeRuns, encodeRuns, type Lattice, rebuildTiles, runsOf, tilesOf} from "./board.ts"
import {latticeIds, latticePositions, latticeSize, positionsHash} from "./lattice.ts"

const STATIC = fileURLToPath(new URL("../../static/", import.meta.url))
const MAP = fileURLToPath(new URL("../../../../map/", import.meta.url))

const lattice = (cols: number): Lattice => ({cols, warp: null, corners: null, order: "three"})

function bytesOf(path: string): ArrayBuffer {
    const bytes = readFileSync(path)
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
}

describe("the shipped board", () => {
    const directory = `${STATIC}boards/${BOARD.name}/`

    it("rebuilds the positions its manifest hashes", () => {
        const {positions, hash} = rebuildTiles(BOARD.lattice, BOARD.tiles, bytesOf(directory + BOARD.files.runs))
        expect(positions.length).toBe(BOARD.tiles * 3)
        expect(hash).toBe(BOARD.positionsHash)
        expect(positionsHash(positions)).toBe(BOARD.positionsHash)
    })

    it("rebuilds every tile of the coordinates blob it replaced bit for bit, in tile order", () => {
        const blob = new Uint32Array(bytesOf(`${MAP}coordinates-10d1f87d.bin`), 12, BOARD.tiles * 3)
        const {positions} = rebuildTiles(BOARD.lattice, BOARD.tiles, bytesOf(directory + BOARD.files.runs))
        const got = new Uint32Array(positions.buffer)
        let differing = 0
        for (let k = 0; k < blob.length; k++) if (got[k] !== blob[k]) differing++
        expect(differing).toBe(0)
    })

    it("is the page's copy of /map's, served where the page fetches it", () => {
        const manifest = readdirSync(directory).filter((name) => /^board-[0-9a-f]{8}\.json$/.test(name))
        expect(manifest).toHaveLength(1)
        const {name, files} = JSON.parse(readFileSync(directory + manifest[0], "utf8"))
        expect(name).toBe(BOARD.name)
        expect(files).toEqual(BOARD.files)
        for (const [kind, file] of Object.entries(BOARD.files)) {
            expect(BOARD.urls[kind as keyof typeof BOARD.urls]).toBe(`/static/boards/${BOARD.name}/${file}`)
            expect(readFileSync(directory + file).equals(readFileSync(`${MAP}boards/${BOARD.name}/${file}`)), file).toBe(true)
        }
    })
})

describe("tilesOf", () => {
    const cols = 5
    const ids = latticeIds(cols)
    const all = latticePositions(cols)
    const kept = [3, 4, 5, 17, 40, 41]
    const runs = runsOf(kept.map((point) => ids[point]))

    it("keeps the points the runs hold, in three's order, with their canonical ids", () => {
        const tiles = tilesOf(lattice(cols), runs, kept.length)

        expect(Array.from(tiles.points)).toEqual(kept)
        expect(Array.from(tiles.ids)).toEqual(kept.map((point) => ids[point]))
        kept.forEach((point, tile) => {
            expect(Array.from(tiles.positions.subarray(tile * 3, tile * 3 + 3))).toEqual(Array.from(all.subarray(point * 3, point * 3 + 3)))
        })
    })

    it("refuses runs that hold another number of tiles than the board says", () => {
        expect(() => tilesOf(lattice(cols), runs, kept.length - 1)).toThrow(/more than/)
        expect(() => tilesOf(lattice(cols), runs, kept.length + 1)).toThrow(/the board says/)
    })

    it("refuses a run past the lattice", () => {
        expect(() => tilesOf(lattice(cols), [[0, latticeSize(cols)]], 1)).toThrow(/past the lattice/)
    })

    it("refuses what is not built yet", () => {
        expect(() => tilesOf({...lattice(cols), order: "canonical"} as unknown as Lattice, runs, kept.length)).toThrow(/order canonical/)
        expect(() => tilesOf({...lattice(cols), warp: {}} as unknown as Lattice, runs, kept.length)).toThrow(/warped/)
    })
})

describe("rebuildTiles", () => {
    it("hands back the cells of the tiles it rebuilt, and their hash", () => {
        const cols = 5
        const ids = latticeIds(cols)
        const kept = [0, 2, 9, 60]
        const runs = encodeRuns(runsOf(kept.map((point) => ids[point])))

        const {positions, cells, hash} = rebuildTiles(lattice(cols), kept.length, runs)

        expect(hash).toBe(positionsHash(positions))
        expect(cells.cols).toBe(cols)
        expect(cells.texels.length).toBe(kept.length)
    })
})

describe("runsOf", () => {
    it("joins ids side by side, in any order given", () => {
        expect(runsOf([7, 3, 4, 5, 9, 10])).toEqual([[3, 5], [7, 7], [9, 10]])
    })

    it("refuses an id twice", () => {
        expect(() => runsOf([1, 2, 2])).toThrow(/twice/)
    })
})

describe("encodeRuns and decodeRuns", () => {
    const runs = [[0, 0], [2, 9], [2 ** 32 + 5, 2 ** 33]] as const

    it("round-trip, ids past 32 bits included", () => {
        expect(decodeRuns(encodeRuns(runs))).toEqual(runs)
    })

    it("write the header and 16 bytes a run", () => {
        const buffer = encodeRuns(runs)
        const view = new DataView(buffer)
        expect(buffer.byteLength).toBe(12 + 16 * runs.length)
        expect(String.fromCharCode(...new Uint8Array(buffer, 0, 4))).toBe("CPRN")
        expect([view.getUint32(4, true), view.getUint32(8, true)]).toEqual([1, runs.length])
    })

    it("refuse a file that is not runs", () => {
        const buffer = encodeRuns(runs)
        new DataView(buffer).setUint8(0, 0)
        expect(() => decodeRuns(buffer)).toThrow(/not a runs file/)
    })

    it("refuse another format version", () => {
        const buffer = encodeRuns(runs)
        new DataView(buffer).setUint32(4, 2, true)
        expect(() => decodeRuns(buffer)).toThrow(/version 2/)
    })

    it("refuse a truncated file", () => {
        const buffer = encodeRuns(runs)
        expect(() => decodeRuns(buffer.slice(0, buffer.byteLength - 1))).toThrow(/size mismatch/)
        expect(() => decodeRuns(buffer.slice(0, 11))).toThrow(/truncated/)
    })

    it("refuse runs out of order or overlapping", () => {
        expect(() => encodeRuns([[5, 6], [1, 2]])).toThrow(/not after/)
        expect(() => encodeRuns([[1, 4], [4, 6]])).toThrow(/not after/)
        expect(() => encodeRuns([[3, 2]])).toThrow(/not a range/)
    })
})
