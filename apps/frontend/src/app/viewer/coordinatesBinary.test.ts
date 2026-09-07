import {describe, expect, it} from "vitest"
import {
    COORDINATES_FORMAT_VERSION,
    COORDINATES_HEADER_BYTES,
    type Coordinates,
    decodeCoordinates,
    encodeCoordinates,
} from "./coordinatesBinary.ts"

function sample(size: number): Coordinates {
    const positions = new Float32Array(size * 3)
    const uvs = new Float32Array(size * 2)
    for (let i = 0; i < size * 3; i++) positions[i] = (i + 1) * 0.25
    for (let i = 0; i < size * 2; i++) uvs[i] = i / (size * 2)
    return {positions, uvs, length: size}
}

describe("encodeCoordinates", () => {
    it("writes the header followed by both float sections", () => {
        const buffer = encodeCoordinates(sample(3))
        expect(buffer.byteLength).toBe(COORDINATES_HEADER_BYTES + 3 * 5 * 4)

        const view = new DataView(buffer)
        expect(String.fromCharCode(...new Uint8Array(buffer, 0, 4))).toBe("CPCO")
        expect(view.getUint32(4, true)).toBe(COORDINATES_FORMAT_VERSION)
        expect(view.getUint32(8, true)).toBe(3)
    })

    it("rejects a tile count that is not a positive integer", () => {
        expect(() => encodeCoordinates({...sample(1), length: 0})).toThrow(/invalid tile count/)
        expect(() => encodeCoordinates({...sample(1), length: -1})).toThrow(/invalid tile count/)
        expect(() => encodeCoordinates({...sample(1), length: 1.5})).toThrow(/invalid tile count/)
    })

    it("rejects sections that do not match the declared tile count", () => {
        expect(() => encodeCoordinates({...sample(2), positions: new Float32Array(5)}))
            .toThrow(/expected 6 position values/)
        expect(() => encodeCoordinates({...sample(2), uvs: new Float32Array(3)}))
            .toThrow(/expected 4 uv values/)
    })

    it("accepts plain arrays, which is what the generator scripts produce", () => {
        const buffer = encodeCoordinates({positions: [1, 2, 3], uvs: [0.5, 0.25], length: 1})
        const decoded = decodeCoordinates(buffer)
        expect(Array.from(decoded.positions)).toEqual([1, 2, 3])
        expect(Array.from(decoded.uvs)).toEqual([0.5, 0.25])
    })
})

describe("decodeCoordinates", () => {
    it("round-trips values that are exact in float32", () => {
        const original = sample(64)
        const decoded = decodeCoordinates(encodeCoordinates(original))

        expect(decoded.size).toBe(64)
        expect(Array.from(decoded.positions)).toEqual(Array.from(original.positions as Float32Array))
        expect(Array.from(decoded.uvs)).toEqual(Array.from(original.uvs as Float32Array))
    })

    it("views the fetched buffer instead of copying it", () => {
        const buffer = encodeCoordinates(sample(4))
        const decoded = decodeCoordinates(buffer)
        expect(decoded.positions.buffer).toBe(buffer)
        expect(decoded.uvs.buffer).toBe(buffer)
    })

    it("rejects a buffer too short to hold a header", () => {
        expect(() => decodeCoordinates(new ArrayBuffer(8))).toThrow(/truncated/)
    })

    it("rejects a file that is not a coordinates blob", () => {
        const buffer = encodeCoordinates(sample(1))
        new DataView(buffer).setUint8(0, "X".charCodeAt(0))
        expect(() => decodeCoordinates(buffer)).toThrow(/not a coordinates file/)
    })

    it("rejects a format version it does not understand", () => {
        const buffer = encodeCoordinates(sample(1))
        new DataView(buffer).setUint32(4, COORDINATES_FORMAT_VERSION + 1, true)
        expect(() => decodeCoordinates(buffer)).toThrow(/unsupported coordinates format version/)
    })

    it("rejects a file declaring zero tiles", () => {
        const buffer = encodeCoordinates(sample(1))
        new DataView(buffer).setUint32(8, 0, true)
        expect(() => decodeCoordinates(buffer)).toThrow(/zero tiles/)
    })

    /**
     * A truncated download is the realistic failure here: the header still
     * parses, so without the length check the geometry would silently come out
     * short rather than the fetch reporting a problem.
     */
    it("rejects a file whose length disagrees with its tile count", () => {
        const buffer = encodeCoordinates(sample(2))
        new DataView(buffer).setUint32(8, 3, true)
        expect(() => decodeCoordinates(buffer)).toThrow(/size mismatch/)
    })
})
