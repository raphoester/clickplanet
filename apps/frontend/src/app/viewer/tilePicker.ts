import * as THREE from "three"
import {latticeFaces, type LatticeFace, tileAt, type TileCells} from "../../domain/latticeCells.ts"

export class TilePicker {
    private readonly at = new THREE.Vector3()
    private readonly ahead = new THREE.Vector3()

    constructor(
        private readonly canvas: Pick<HTMLCanvasElement, "width" | "height">,
        private readonly cells: TileCells,
        private readonly faces: readonly LatticeFace[] = latticeFaces(cells.lattice),
    ) {
    }

    // x and y in drawing-buffer pixels; undefined off the globe or over the sea.
    pick(camera: THREE.OrthographicCamera, x: number, y: number): number | undefined {
        const {width, height} = this.canvas
        if (x < 0 || y < 0 || x >= width || y >= height) return undefined

        camera.updateMatrixWorld()
        const from = this.at.set((Math.floor(x) + 0.5) / width * 2 - 1, -((Math.floor(y) + 0.5) / height) * 2 + 1, -1).unproject(camera)
        const ahead = this.ahead.set(0, 0, -1).transformDirection(camera.matrixWorld)

        // Orthographic: every pixel's ray runs along the camera's axis, onto the tiles' sphere of radius 1.
        const along = from.dot(ahead)
        const inside = along * along - from.lengthSq() + 1
        if (inside < 0) return undefined
        const ground = from.addScaledVector(ahead, -along - Math.sqrt(inside))
        return tileAt(this.cells, this.faces, ground.x, ground.y, ground.z)
    }
}
