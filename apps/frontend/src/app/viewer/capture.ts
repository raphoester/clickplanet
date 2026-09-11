import * as THREE from "three";

export type CapturedFrame = {
    /** RGBA, top row first, ready for `ImageData`. */
    pixels: Uint8ClampedArray
    width: number
    height: number
}

/**
 * The frame the player is looking at, read straight out of the drawing buffer.
 *
 * **This must run in the same tick as the `render()` that filled it.** The
 * renderer is built without `preserveDrawingBuffer` — see `setupScene` for why —
 * so the buffer is gone as soon as control goes back to the browser, and a read
 * one tick late comes back blank. `globe.ts` calls it from inside the animation
 * loop, immediately after the render, which is the only place that holds.
 *
 * Reading the default framebuffer rather than an offscreen target is also what
 * keeps the capture honest: three applies the output colour space conversion
 * only when it renders to the canvas, so the same scene drawn into a render
 * target comes back in linear sRGB — a visibly different image from the one the
 * player is being offered.
 */
export function readDrawingBuffer(renderer: THREE.WebGLRenderer): CapturedFrame {
    const gl = renderer.getContext()
    const {width, height} = renderer.domElement

    // Uint8Array rather than the clamped one the caller wants: WebGL 1 accepts
    // only the former, and the two share a buffer, so the wrap costs nothing.
    const pixels = new Uint8Array(width * height * 4)
    gl.readPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, pixels)

    return {pixels: flipRows(new Uint8ClampedArray(pixels.buffer), width, height), width, height}
}

/** GL hands back the bottom row first; `ImageData` wants the top one. */
export function flipRows(pixels: Uint8ClampedArray, width: number, height: number): Uint8ClampedArray {
    const stride = width * 4
    const flipped = new Uint8ClampedArray(pixels.length)

    for (let row = 0; row < height; row++) {
        flipped.set(pixels.subarray(row * stride, (row + 1) * stride), (height - 1 - row) * stride)
    }

    return flipped
}
