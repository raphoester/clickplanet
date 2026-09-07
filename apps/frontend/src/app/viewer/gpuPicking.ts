import * as THREE from "three"
import {colorToInteger} from "./pickingColors.ts";
import {innerSphere} from "./sphere.ts";
import {pickWindowSize, tilePointSize} from "./pointSize.ts";

/** Read back from an empty buffer; not a tile. */
const BLANK = colorToInteger([255, 255, 255])

/** The background and the occluding sphere; not a tile. */
const BACKGROUND = colorToInteger([0, 0, 0])

/**
 * Resolves a screen position to the tile under it, by rendering the globe once
 * with each tile in its own colour and reading the pixel back.
 *
 * The expensive part is that the read is synchronous: it stalls the GPU
 * pipeline until the render it depends on has finished. That is unavoidable
 * here, so everything around it is arranged to make it as cheap as possible.
 *
 * This used to run per mousemove event and, each time, allocate a full-window
 * render target, build a Scene, and construct a Mesh and a MeshBasicMaterial
 * that were never disposed — then render all ~258k points across every pixel
 * on screen to read exactly one of them. The scene and the target are built
 * once now, and `setViewOffset` narrows the projection to a small window around
 * the cursor, so the render covers a few dozen pixels rather than the viewport.
 *
 * That window cannot be a single pixel. A tile is drawn as a point sprite
 * `pointSize` across, so it covers the cursor's pixel while its own centre sits
 * up to half that away — and a point whose centre falls outside the rendered
 * window is clipped before it can draw anything. Rendering 1x1 therefore lost
 * every tile the cursor was not dead-centre on: 29% of clicks on land picked
 * nothing, and another 13% picked a neighbour.
 */
export class GpuPicker {
    private readonly scene = new THREE.Scene()
    /** Resized only when the zoom changes the window it needs, not per pick. */
    private readonly target = new THREE.WebGLRenderTarget(3, 3)
    private readonly occluder: THREE.Mesh
    private readonly pixel = new Uint8Array(4)

    constructor(private readonly renderer: THREE.WebGLRenderer, pickingPoints: THREE.Points) {
        /**
         * Black, matching the background: tiles on the far side of the globe
         * are hidden behind it and read back as "nothing", so the cursor cannot
         * pick a tile it is pointing through the planet at.
         */
        this.occluder = new THREE.Mesh(innerSphere(), new THREE.MeshBasicMaterial({color: 0x000000}))

        this.scene.add(pickingPoints)
        this.scene.add(this.occluder)
    }

    /**
     * The tile at a position in CSS pixels relative to the canvas, or undefined
     * for the background, the sphere, or a position off-canvas.
     */
    pick(camera: THREE.OrthographicCamera, x: number, y: number): number | undefined {
        const {width, height} = this.renderer.domElement
        if (x < 0 || y < 0 || x >= width || y >= height) return undefined

        const size = pickWindowSize(tilePointSize(camera.zoom, height))
        const middle = (size - 1) / 2

        /**
         * One target pixel per screen pixel. Scaling the window onto a
         * differently sized target would leave the sprites the wrong size
         * relative to it, which is the same way a 1x1 window failed.
         */
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
