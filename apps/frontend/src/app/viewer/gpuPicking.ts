import * as THREE from "three"
import {colorToInteger} from "./pickingColors.ts";
import {pickPointSize, pickWindowSize} from "./pointSize.ts";

const BLANK = colorToInteger([255, 255, 255])

const BACKGROUND = colorToInteger([0, 0, 0])

export class GpuPicker {
    private readonly scene = new THREE.Scene()
    private readonly target = new THREE.WebGLRenderTarget(3, 3)
    private readonly occluder: THREE.Mesh
    private readonly pixel = new Uint8Array(4)
    private readonly clearColour = new THREE.Color()

    constructor(private readonly renderer: THREE.WebGLRenderer, pickingPoints: THREE.Points, sphere: THREE.BufferGeometry) {
        this.occluder = new THREE.Mesh(sphere, new THREE.MeshBasicMaterial({color: 0x000000}))

        this.scene.add(pickingPoints)
        this.scene.add(this.occluder)
    }

    pick(camera: THREE.OrthographicCamera, x: number, y: number): number | undefined {
        const {width, height} = this.renderer.domElement
        if (x < 0 || y < 0 || x >= width || y >= height) return undefined

        const ratio = this.renderer.getPixelRatio()
        const size = pickWindowSize(pickPointSize(camera.zoom, height / ratio) * ratio)
        const middle = (size - 1) / 2

        if (this.target.width !== size) this.target.setSize(size, size)

        const previousTarget = this.renderer.getRenderTarget()
        this.renderer.getClearColor(this.clearColour)
        const clearAlpha = this.renderer.getClearAlpha()

        camera.setViewOffset(width, height, Math.floor(x) - middle, Math.floor(y) - middle, size, size)
        this.renderer.setRenderTarget(this.target)
        // Black is no tile: the sky's colour would read as one.
        this.renderer.setClearColor(0x000000, 1)
        this.renderer.render(this.scene, camera)
        this.renderer.readRenderTargetPixels(this.target, middle, middle, 1, 1, this.pixel)

        this.renderer.setRenderTarget(previousTarget)
        // The canvas's clear takes the last colour handed to the GPU: given while on the target, the sky came out black.
        this.renderer.setClearColor(this.clearColour, clearAlpha)
        camera.clearViewOffset()

        const id = colorToInteger([this.pixel[0], this.pixel[1], this.pixel[2]])
        return id === BACKGROUND || id === BLANK ? undefined : id
    }

    dispose() {
        this.scene.clear()
        this.target.dispose()
        ;(this.occluder.material as THREE.Material).dispose()
    }
}
