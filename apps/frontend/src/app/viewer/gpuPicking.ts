import * as THREE from "three"
import {colorToInteger} from "./pickingColors.ts";
import {innerSphere} from "./sphere.ts";
import {pickWindowSize, tilePointSize} from "./pointSize.ts";

const BLANK = colorToInteger([255, 255, 255])

const BACKGROUND = colorToInteger([0, 0, 0])

export class GpuPicker {
    private readonly scene = new THREE.Scene()
    private readonly target = new THREE.WebGLRenderTarget(3, 3)
    private readonly occluder: THREE.Mesh
    private readonly pixel = new Uint8Array(4)

    constructor(private readonly renderer: THREE.WebGLRenderer, pickingPoints: THREE.Points) {
        this.occluder = new THREE.Mesh(innerSphere(), new THREE.MeshBasicMaterial({color: 0x000000}))

        this.scene.add(pickingPoints)
        this.scene.add(this.occluder)
    }

    pick(camera: THREE.OrthographicCamera, x: number, y: number): number | undefined {
        const {width, height} = this.renderer.domElement
        if (x < 0 || y < 0 || x >= width || y >= height) return undefined

        const size = pickWindowSize(tilePointSize(camera.zoom, height))
        const middle = (size - 1) / 2

        if (this.target.width !== size) this.target.setSize(size, size)

        const previousTarget = this.renderer.getRenderTarget()

        camera.setViewOffset(width, height, Math.floor(x) - middle, Math.floor(y) - middle, size, size)
        this.renderer.setRenderTarget(this.target)
        this.renderer.render(this.scene, camera)
        this.renderer.readRenderTargetPixels(this.target, middle, middle, 1, 1, this.pixel)

        this.renderer.setRenderTarget(previousTarget)
        camera.clearViewOffset()

        const id = colorToInteger([this.pixel[0], this.pixel[1], this.pixel[2]])
        return id === BACKGROUND || id === BLANK ? undefined : id
    }

    dispose() {
        this.scene.clear()
        this.target.dispose()
        this.occluder.geometry.dispose()
        ;(this.occluder.material as THREE.Material).dispose()
    }
}
