/**
 * Binary container for the tile coordinates.
 *
 * The map is ~258k tiles, which is ~25 MB as JSON but only ~5 MB as float32.
 * Shipping it as a binary blob fetched at runtime keeps it out of the JS chunk
 * (Cloudflare Pages rejects any single file above 25 MiB) and lets the decoder
 * hand the bytes straight to Three.js instead of re-copying a parsed array.
 *
 * Layout, all little-endian:
 *
 *   offset  0   4 bytes    magic "CPCO"
 *   offset  4   uint32     format version
 *   offset  8   uint32     tile count N
 *   offset 12   N*3 f32    positions (x, y, z per tile)
 *   offset 12+N*12  N*2 f32 uvs (u, v per tile)
 *
 * The 12-byte header keeps the float sections 4-byte aligned, so the decoder can
 * take zero-copy Float32Array views over the fetched ArrayBuffer.
 */

export const COORDINATES_MAGIC = "CPCO"
export const COORDINATES_FORMAT_VERSION = 1
export const COORDINATES_HEADER_BYTES = 12

/** Floats per tile: 3 for the position, 2 for the uv. */
const FLOATS_PER_TILE = 5

export type Coordinates = {
    positions: ArrayLike<number>
    uvs: ArrayLike<number>
    length: number
}

export type PointGeometryData = {
    positions: Float32Array
    uvs: Float32Array
    size: number
}

const littleEndian = new Uint8Array(new Uint16Array([1]).buffer)[0] === 1

export function encodeCoordinates(coordinates: Coordinates): ArrayBuffer {
    const size = coordinates.length
    if (!Number.isInteger(size) || size <= 0) {
        throw new Error(`invalid tile count: ${size}`)
    }
    if (coordinates.positions.length !== size * 3) {
        throw new Error(`expected ${size * 3} position values, got ${coordinates.positions.length}`)
    }
    if (coordinates.uvs.length !== size * 2) {
        throw new Error(`expected ${size * 2} uv values, got ${coordinates.uvs.length}`)
    }

    const buffer = new ArrayBuffer(COORDINATES_HEADER_BYTES + size * FLOATS_PER_TILE * 4)
    const view = new DataView(buffer)

    for (let i = 0; i < COORDINATES_MAGIC.length; i++) {
        view.setUint8(i, COORDINATES_MAGIC.charCodeAt(i))
    }
    view.setUint32(4, COORDINATES_FORMAT_VERSION, true)
    view.setUint32(8, size, true)

    let offset = COORDINATES_HEADER_BYTES
    for (let i = 0; i < size * 3; i++, offset += 4) {
        view.setFloat32(offset, coordinates.positions[i], true)
    }
    for (let i = 0; i < size * 2; i++, offset += 4) {
        view.setFloat32(offset, coordinates.uvs[i], true)
    }

    return buffer
}

export function decodeCoordinates(buffer: ArrayBuffer): PointGeometryData {
    if (buffer.byteLength < COORDINATES_HEADER_BYTES) {
        throw new Error(`coordinates file is truncated: ${buffer.byteLength} bytes`)
    }

    const view = new DataView(buffer)
    let magic = ""
    for (let i = 0; i < COORDINATES_MAGIC.length; i++) {
        magic += String.fromCharCode(view.getUint8(i))
    }
    if (magic !== COORDINATES_MAGIC) {
        throw new Error(`not a coordinates file: expected magic "${COORDINATES_MAGIC}", got "${magic}"`)
    }

    const version = view.getUint32(4, true)
    if (version !== COORDINATES_FORMAT_VERSION) {
        throw new Error(`unsupported coordinates format version ${version}, expected ${COORDINATES_FORMAT_VERSION}`)
    }

    const size = view.getUint32(8, true)
    const expectedBytes = COORDINATES_HEADER_BYTES + size * FLOATS_PER_TILE * 4
    if (size === 0) {
        throw new Error("coordinates file declares zero tiles")
    }
    if (buffer.byteLength !== expectedBytes) {
        throw new Error(
            `coordinates file size mismatch: ${size} tiles need ${expectedBytes} bytes, got ${buffer.byteLength}`
        )
    }

    const positionsOffset = COORDINATES_HEADER_BYTES
    const uvsOffset = positionsOffset + size * 3 * 4

    return {
        positions: readFloats(buffer, positionsOffset, size * 3),
        uvs: readFloats(buffer, uvsOffset, size * 2),
        size,
    }
}

function readFloats(buffer: ArrayBuffer, byteOffset: number, count: number): Float32Array {
    // Typed-array views inherit the platform's byte order, so they are only a
    // valid read of a little-endian file on a little-endian host. Every browser
    // target is little-endian; the DataView path is the correctness fallback.
    if (littleEndian) return new Float32Array(buffer, byteOffset, count)

    const view = new DataView(buffer)
    const floats = new Float32Array(count)
    for (let i = 0; i < count; i++) {
        floats[i] = view.getFloat32(byteOffset + i * 4, true)
    }
    return floats
}
