// The provinces' lines of a board, as `npm run borderLines` writes them beside the countries' and the globe reads them
// (the page's docs/country-outlines.md, /map/README.md#format).
import {type BorderLineData, decodeBorderLines, encodeBorderLines} from "./borderLinesFile.ts"

export type ProvinceLineData = {
    // The countries' lines and the provinces these were traced with: the 8 hex of each file's name, as a number.
    countryLines: number
    provinces: number
    // The cell edges between two provinces of one country, chained into runs, each with its two provinces.
    lines: BorderLineData
    // The two provinces either side of each piece of the countries' lines, in their order, the sea 0: the first on the
    // side of the piece's run's first landmass.
    countrySides: Uint16Array
}

const MAGIC = "CPPL"
const VERSION = 1
const HEAD_BYTES = 28

// The countries' pieces' sides change only where a province ends along a run: they are written as spans, each its pair
// and then its count of pieces as a varint.
export function encodeProvinceLines({countryLines, provinces, lines, countrySides}: ProvinceLineData): Uint8Array {
    const pieces = countrySides.length / 2
    const pairs: number[] = []
    const counts: number[] = []
    for (let piece = 0; piece < pieces;) {
        const a = countrySides[piece * 2], b = countrySides[piece * 2 + 1]
        let end = piece + 1
        while (end < pieces && countrySides[end * 2] === a && countrySides[end * 2 + 1] === b) end++
        pairs.push(a, b)
        pushVarint(counts, end - piece)
        piece = end
    }

    const blob = encodeBorderLines(lines)
    const bytes = new Uint8Array(HEAD_BYTES + blob.length + pairs.length * 2 + counts.length)
    const head = new DataView(bytes.buffer)
    for (let i = 0; i < MAGIC.length; i++) head.setUint8(i, MAGIC.charCodeAt(i))
    head.setUint32(4, VERSION, true)
    head.setUint32(8, countryLines, true)
    head.setUint32(12, provinces, true)
    head.setUint32(16, pieces, true)
    head.setUint32(20, pairs.length / 2, true)
    head.setUint32(24, blob.length, true)
    bytes.set(blob, HEAD_BYTES)
    let at = HEAD_BYTES + blob.length
    for (const value of pairs) {
        head.setUint16(at, value, true)
        at += 2
    }
    bytes.set(counts, at)
    return bytes
}

export function decodeProvinceLines(buffer: ArrayBuffer): ProvinceLineData {
    if (buffer.byteLength < HEAD_BYTES) throw new Error(`province-line blob is truncated: ${buffer.byteLength} bytes`)
    const head = new DataView(buffer)
    const magic = String.fromCharCode(head.getUint8(0), head.getUint8(1), head.getUint8(2), head.getUint8(3))
    if (magic !== MAGIC) throw new Error(`not a province-line blob: magic "${magic}"`)
    const version = head.getUint32(4, true)
    if (version !== VERSION) throw new Error(`province-line blob version ${version}, expected ${VERSION}`)

    const pieces = head.getUint32(16, true)
    const spans = head.getUint32(20, true)
    const blobBytes = head.getUint32(24, true)
    const pairsAt = HEAD_BYTES + blobBytes
    const countsAt = pairsAt + spans * 4
    if (buffer.byteLength < countsAt) throw new Error(`province-line blob is truncated: ${spans} spans need ${countsAt} bytes`)
    const lines = decodeBorderLines(buffer.slice(HEAD_BYTES, pairsAt))

    const countrySides = new Uint16Array(pieces * 2)
    const counts = new Uint8Array(buffer, countsAt)
    let read = 0
    let piece = 0
    for (let span = 0; span < spans; span++) {
        let count = 0
        let shift = 0
        let byte: number
        do {
            if (read === counts.length) throw new Error("province-line blob is truncated in its spans")
            byte = counts[read++]
            count |= (byte & 0x7f) << shift
            shift += 7
        } while (byte & 0x80)
        if (piece + count > pieces) throw new Error(`span ${span} runs past the countries' ${pieces} pieces`)
        const a = head.getUint16(pairsAt + span * 4, true), b = head.getUint16(pairsAt + span * 4 + 2, true)
        for (const end = piece + count; piece < end; piece++) {
            countrySides[piece * 2] = a
            countrySides[piece * 2 + 1] = b
        }
    }
    if (piece !== pieces) throw new Error(`the spans hold ${piece} pieces, the blob says ${pieces}`)
    if (read !== counts.length) throw new Error(`${counts.length - read} bytes past the last span`)
    return {countryLines: head.getUint32(8, true), provinces: head.getUint32(12, true), lines, countrySides}
}

// The 8 hex of a content-addressed file's name, as a number: "borderLines-0cd29f62.bin" is 0x0cd29f62.
export function fileHashOf(name: string): number {
    const hash = /-([0-9a-f]{8})\.[a-z]+$/.exec(name)?.[1]
    if (hash === undefined) throw new Error(`${name} is not a content-addressed file's name`)
    return parseInt(hash, 16)
}

function pushVarint(out: number[], value: number): void {
    while (value >= 0x80) {
        out.push((value & 0x7f) | 0x80)
        value >>>= 7
    }
    out.push(value)
}
