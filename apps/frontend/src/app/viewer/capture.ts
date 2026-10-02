import * as THREE from "three";

export type CapturedFrame = {
    pixels: Uint8ClampedArray
    width: number
    height: number
}

// Call in the same tick as render(): without preserveDrawingBuffer the frame is gone after.
export function readDrawingBuffer(renderer: THREE.WebGLRenderer): CapturedFrame {
    const gl = renderer.getContext()
    const {width, height} = renderer.domElement

    const pixels = new Uint8Array(width * height * 4)
    gl.readPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, pixels)

    return {pixels: flipRows(new Uint8ClampedArray(pixels.buffer), width, height), width, height}
}

export function flipRows(pixels: Uint8ClampedArray, width: number, height: number): Uint8ClampedArray {
    const stride = width * 4
    const flipped = new Uint8ClampedArray(pixels.length)

    for (let row = 0; row < height; row++) {
        flipped.set(pixels.subarray(row * stride, (row + 1) * stride), (height - 1 - row) * stride)
    }

    return flipped
}
