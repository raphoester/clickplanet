import {describe, expect, it} from "vitest"
import {type BorderLineData, decodeBorderLines, encodeBorderLines} from "./borderLinesFile.ts"
import {BOARDS} from "../app/viewer/boardsAsset.ts"
import {bytesOf} from "../testdata/boards.ts"

function lines(runs: number[][], sides: number[] = runs.flatMap(() => [0, 1])): BorderLineData {
    return {runs: Uint32Array.from(runs, (run) => run.length / 3), sides: Uint16Array.from(sides), corners: Int16Array.from(runs.flat())}
}

function bufferOf(bytes: Uint8Array): ArrayBuffer {
    return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
}

function plain({runs, sides, corners}: BorderLineData) {
    return {runs: [...runs], sides: [...sides], corners: [...corners]}
}

// FNV-1a over the bytes of the runs, the sides and the corners, in turn.
function fingerprint({runs, sides, corners}: BorderLineData): string {
    let hash = 0x811c9dc5
    for (const array of [runs, sides, corners]) {
        for (const byte of new Uint8Array(array.buffer, array.byteOffset, array.byteLength)) hash = Math.imul(hash ^ byte, 0x01000193) >>> 0
    }
    return hash.toString(16).padStart(8, "0")
}

describe("the border lines' file", () => {
    it("reads back the runs, the landmasses either side of each and the corners", () => {
        const written = lines([[1, 2, 3, 4, 5, 6], [7, 8, 9, 10, 11, 12, 13, 14, 15]], [0, 4, 4, 9])
        expect(plain(decodeBorderLines(bufferOf(encodeBorderLines(written))))).toEqual(plain(written))
    })

    it("reads back corners as far apart as the int16s go, and a run of one corner", () => {
        const written = lines([[-32767, 32767, 0, 32767, -32767, -1, 0, 0, 0, 5, -5, 300], [12, -12, 7]])
        expect(plain(decodeBorderLines(bufferOf(encodeBorderLines(written))))).toEqual(plain(written))
    })

    it("takes a byte a step between two corners a cell edge apart", () => {
        const step = encodeBorderLines(lines([[1000, 2000, 3000, 1040, 1950, 3010]])).length
        const still = encodeBorderLines(lines([[1000, 2000, 3000]])).length
        expect(step - still).toBe(3)
    })

    it("refuses a blob that is not one", () => {
        const bytes = encodeBorderLines(lines([[1, 2, 3, 4, 5, 6]]))
        bytes[0] = "X".charCodeAt(0)
        expect(() => decodeBorderLines(bufferOf(bytes))).toThrow(/magic/)
    })

    it("refuses version 2, whose corners were plain int16s", () => {
        const bytes = encodeBorderLines(lines([[1, 2, 3, 4, 5, 6]]))
        new DataView(bytes.buffer).setUint32(4, 2, true)
        expect(() => decodeBorderLines(bufferOf(bytes))).toThrow(/version 2/)
    })

    it("refuses a blob cut short, and one with bytes past its last corner", () => {
        const bytes = encodeBorderLines(lines([[1, 2, 3, 400, 500, 600]]))
        expect(() => decodeBorderLines(bufferOf(bytes.slice(0, bytes.length - 1)))).toThrow(/truncated/)
        expect(() => decodeBorderLines(bufferOf(Uint8Array.from([...bytes, 0])))).toThrow(/past the last corner/)
    })

    // What version 2 held of each board, its corners as plain int16s (2026-11-18): version 3 reads back the same lines.
    const VERSION_2: Record<string, string> = {"earth-1": "df71f4ea", "earth-2": "1679628d", "siege-1": "3ccffc17"}

    for (const board of Object.values(BOARDS)) {
        it(`reads ${board.name}'s outline as version 2 held it, and writes it back to the same bytes`, () => {
            const shipped = bytesOf(board, board.page.borderLines)
            const decoded = decodeBorderLines(shipped)
            expect(fingerprint(decoded)).toBe(VERSION_2[board.name])
            expect(Buffer.from(encodeBorderLines(decoded)).equals(Buffer.from(shipped))).toBe(true)
        })
    }
})
