import {describe, expect, it} from "vitest"
import {decodeBorderLines} from "./borderLinesFile.ts"
import {decodeProvinceLines, encodeProvinceLines, fileHashOf, type ProvinceLineData} from "./provinceLinesFile.ts"
import {decodeProvinces} from "./provinces.ts"
import {pieceSides} from "../app/viewer/borderLines.ts"
import {BOARDS} from "../app/viewer/boardsAsset.ts"
import {bytesOf} from "../testdata/boards.ts"

function bufferOf(bytes: Uint8Array): ArrayBuffer {
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
}

const WRITTEN: ProvinceLineData = {
    countryLines: 0x0cd29f62,
    provinces: 0x286f4c3e,
    lines: {runs: Uint32Array.from([2, 3]), sides: Uint16Array.from([1, 2, 2, 3]), corners: Int16Array.from([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15])},
    countrySides: Uint16Array.from([0, 1, 0, 1, 0, 2, 4, 5, 4, 5, 4, 5]),
}

function plain({countryLines, provinces, lines, countrySides}: ProvinceLineData) {
    return {countryLines, provinces, runs: [...lines.runs], sides: [...lines.sides], corners: [...lines.corners], countrySides: [...countrySides]}
}

describe("the province lines' file", () => {
    it("reads back the lines, the files they pair with and the provinces beside each piece of the countries'", () => {
        expect(plain(decodeProvinceLines(bufferOf(encodeProvinceLines(WRITTEN))))).toEqual(plain(WRITTEN))
    })

    it("writes a span for each change of pair along the countries' pieces", () => {
        const one = encodeProvinceLines({...WRITTEN, countrySides: Uint16Array.from([4, 5, 4, 5, 4, 5, 4, 5])}).length
        const two = encodeProvinceLines({...WRITTEN, countrySides: Uint16Array.from([4, 5, 4, 5, 4, 6, 4, 6])}).length
        expect(two - one).toBe(5)
    })

    it("reads a span of more pieces than a byte counts", () => {
        const written = {...WRITTEN, countrySides: new Uint16Array(600).fill(7)}
        expect(plain(decodeProvinceLines(bufferOf(encodeProvinceLines(written))))).toEqual(plain(written))
    })

    it("refuses a blob that is not one, of another version, cut short, or with bytes past its last span", () => {
        const bytes = encodeProvinceLines(WRITTEN)
        const named = Uint8Array.from(bytes)
        named[0] = "X".charCodeAt(0)
        expect(() => decodeProvinceLines(bufferOf(named))).toThrow(/magic/)
        const later = Uint8Array.from(bytes)
        new DataView(later.buffer).setUint32(4, 2, true)
        expect(() => decodeProvinceLines(bufferOf(later))).toThrow(/version 2/)
        expect(() => decodeProvinceLines(bufferOf(bytes.slice(0, bytes.length - 1)))).toThrow(/truncated/)
        expect(() => decodeProvinceLines(bufferOf(Uint8Array.from([...bytes, 0])))).toThrow(/past the last span/)
    })

    it("reads a content-addressed file's hash from its name", () => {
        expect(fileHashOf("borderLines-0cd29f62.bin")).toBe(0x0cd29f62)
        expect(fileHashOf("provinces-286f4c3e.bin")).toBe(0x286f4c3e)
        expect(() => fileHashOf("borderLines.bin")).toThrow(/content-addressed/)
    })

    for (const board of Object.values(BOARDS)) {
        const {provinceLines} = board.page
        if (board.files.provinces === undefined) {
            it(`gives ${board.name}, which has no provinces, no province lines`, () => {
                expect(provinceLines).toBeUndefined()
            })
            continue
        }
        if (provinceLines === undefined) throw new Error(`${board.name} has provinces and no province lines`)

        it(`pairs ${board.name}'s province lines with its countries' lines and its provinces`, () => {
            const read = decodeProvinceLines(bytesOf(board, provinceLines))
            expect(read.countryLines).toBe(fileHashOf(board.page.borderLines))
            expect(read.provinces).toBe(fileHashOf(board.files.provinces!))
        })

        it(`draws ${board.name}'s province lines between two provinces of one landmass only`, () => {
            const {lines} = decodeProvinceLines(bytesOf(board, provinceLines))
            const {landmass} = decodeProvinces(bytesOf(board, board.files.provinces!), board.tiles)
            for (let run = 0; run < lines.runs.length; run++) {
                const [a, b] = [lines.sides[run * 2], lines.sides[run * 2 + 1]]
                expect(a).toBeGreaterThan(0)
                expect(b).toBeGreaterThan(a)
                expect(landmass[a]).toBe(landmass[b])
            }
        })

        it(`puts on each side of ${board.name}'s countries' pieces a province of that side's landmass`, () => {
            const {countrySides} = decodeProvinceLines(bytesOf(board, provinceLines))
            const {landmass} = decodeProvinces(bytesOf(board, board.files.provinces!), board.tiles)
            const sides = pieceSides(decodeBorderLines(bytesOf(board, board.page.borderLines)))
            expect(countrySides.length).toBe(sides.length)
            const astray = sides.filter((side, at) => (countrySides[at] === 0 ? 0 : landmass[countrySides[at]]) !== side).length
            expect(astray).toBe(0)
        })
    }
})
