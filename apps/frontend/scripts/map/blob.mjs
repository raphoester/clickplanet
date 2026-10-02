// Reads the two shared blobs from /map, for the tools that only look at them.
//
// The writers stay where they are — `writeCoordinates.ts` runs under vite-node and shares the
// encoder with the app, which is what keeps the two ends of the format in one place. These readers
// exist because the audit is plain node and only needs to look.
import fs from "node:fs"
import path from "node:path"
import {fileURLToPath} from "node:url"

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..", "..")

export const mapDir = path.resolve(frontendRoot, "..", "..", "map")
export const staticDir = path.join(frontendRoot, "static")

const COORDINATES = /^coordinates-[0-9a-f]{8}\.bin$/
const BORDERS = /^borders-[0-9a-f]{8}\.bin$/

/** The one blob of a kind in a directory. Content-addressed, so a second one is a stale copy. */
export function blobNamed(pattern, directory = mapDir) {
    const names = fs.readdirSync(directory).filter((entry) => pattern.test(entry))
    if (names.length !== 1) {
        throw new Error(`expected one ${directory}/${pattern.source}, found ${names.length}`)
    }
    return names[0]
}

export const coordinatesName = (directory) => blobNamed(COORDINATES, directory)
export const bordersName = (directory) => blobNamed(BORDERS, directory)

/** @returns {{name: string, count: number, positions: Float32Array, uvs: Float32Array}} */
export function readCoordinates(directory = mapDir) {
    const name = coordinatesName(directory)
    const bytes = fs.readFileSync(path.join(directory, name))
    const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)

    let magic = ""
    for (let i = 0; i < 4; i++) magic += String.fromCharCode(view.getUint8(i))
    if (magic !== "CPCO") throw new Error(`${name}: expected magic "CPCO", got "${magic}"`)

    const version = view.getUint32(4, true)
    if (version !== 1) throw new Error(`${name}: unsupported format version ${version}`)

    const count = view.getUint32(8, true)
    const expected = 12 + count * 5 * 4
    if (bytes.byteLength !== expected) {
        throw new Error(`${name}: ${count} tiles need ${expected} bytes, got ${bytes.byteLength}`)
    }

    return {
        name,
        count,
        positions: new Float32Array(bytes.buffer, bytes.byteOffset + 12, count * 3),
        uvs: new Float32Array(bytes.buffer, bytes.byteOffset + 12 + count * 12, count * 2),
    }
}

/** @returns {{name: string, count: number, codes: string[], landmass: Uint16Array}} */
export function readBorders(directory = mapDir) {
    const name = bordersName(directory)
    const bytes = fs.readFileSync(path.join(directory, name))
    const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)

    const headerLength = view.getUint32(0, true)
    const {tiles, codes} = JSON.parse(bytes.subarray(4, 4 + headerLength).toString("utf8"))

    return {
        name,
        count: tiles,
        codes,
        landmass: new Uint16Array(bytes.buffer, bytes.byteOffset + 4 + headerLength, tiles),
    }
}
