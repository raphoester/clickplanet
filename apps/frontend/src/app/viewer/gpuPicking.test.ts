import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {GpuPicker} from "./gpuPicking.ts"

const SKY = new THREE.Color(0x080b1a)

// Records what the picker asks of the renderer, in order: the GPU's clear colour is whatever was set last.
class RecordingRenderer {
    readonly domElement = {width: 800, height: 600}
    readonly calls: string[] = []
    private target: THREE.WebGLRenderTarget | null = null
    private readonly clearColour = SKY.clone()
    private clearAlpha = 1
    rendered = new THREE.Color()

    getPixelRatio() {
        return 1
    }

    getRenderTarget() {
        return this.target
    }

    setRenderTarget(target: THREE.WebGLRenderTarget | null) {
        this.target = target
        this.calls.push(target === null ? "canvas" : "target")
    }

    getClearColor(into: THREE.Color) {
        return into.copy(this.clearColour)
    }

    getClearAlpha() {
        return this.clearAlpha
    }

    setClearColor(colour: THREE.ColorRepresentation, alpha: number) {
        this.clearColour.set(colour)
        this.clearAlpha = alpha
        this.calls.push(`clear ${this.clearColour.getHexString()}`)
    }

    render() {
        this.rendered.copy(this.clearColour)
        this.calls.push("render")
    }

    readRenderTargetPixels(_target: unknown, _x: number, _y: number, _w: number, _h: number, into: Uint8Array) {
        into.set([0, 0, 0, 255])
    }
}

function picker() {
    const renderer = new RecordingRenderer()
    const points = new THREE.Points(new THREE.BufferGeometry(), new THREE.PointsMaterial())
    const gpu = new GpuPicker(renderer as unknown as THREE.WebGLRenderer, points, new THREE.SphereGeometry())
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0.01, 100)
    return {renderer, gpu, camera}
}

describe("GpuPicker", () => {
    it("picks against black, which is no tile, whatever the sky's colour", () => {
        const {renderer, gpu, camera} = picker()

        expect(gpu.pick(camera, 400, 300)).toBeUndefined()
        expect(renderer.rendered.getHex()).toBe(0x000000)
    })

    it("gives the canvas its sky back once it is drawing to it again", () => {
        const {renderer, gpu, camera} = picker()

        gpu.pick(camera, 400, 300)

        expect(renderer.calls).toEqual(["target", "clear 000000", "render", "canvas", `clear ${SKY.getHexString()}`])
    })
})
