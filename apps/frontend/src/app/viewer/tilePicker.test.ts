import {readFileSync} from "node:fs"
import {fileURLToPath} from "node:url"
import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {TilePicker} from "./tilePicker.ts"
import {BOARD} from "./boardAsset.ts"
import {rebuildTiles} from "../../domain/board.ts"

const STATIC = fileURLToPath(new URL("../../../static/", import.meta.url))

const bytes = readFileSync(`${STATIC}boards/${BOARD.name}/${BOARD.files.runs}`)
const {positions, cells} = rebuildTiles(BOARD.lattice, BOARD.tiles, bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength))

const canvas = {width: 1001, height: 1001}
const MIDDLE = 500
const picker = new TilePicker(canvas, cells)

function positionOf(tile: number): THREE.Vector3 {
    return new THREE.Vector3(positions[tile * 3 - 3], positions[tile * 3 - 2], positions[tile * 3 - 1])
}

function cameraOn(direction: THREE.Vector3, zoom: number): THREE.OrthographicCamera {
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0.01, 100)
    camera.position.copy(direction).setLength(5)
    camera.lookAt(0, 0, 0)
    camera.zoom = zoom
    camera.updateProjectionMatrix()
    return camera
}

describe("TilePicker", () => {
    it("finds the tile in the middle of the screen, from orbit and up close", () => {
        for (const tile of [1, 4_321, 98_765, 260_503]) {
            for (const zoom of [1, 10, 50]) {
                expect(picker.pick(cameraOn(positionOf(tile), zoom), MIDDLE, MIDDLE), `tile ${tile} at zoom ${zoom}`).toBe(tile)
            }
        }
    })

    it("takes the gap between two tiles as the nearer one's", () => {
        const tile = 98_765
        const centre = positionOf(tile)
        const aside = new THREE.Vector3(0, 1, 0).cross(centre).normalize().multiplyScalar(0.0039 * 0.4)
        expect(picker.pick(cameraOn(centre.clone().add(aside), 50), MIDDLE, MIDDLE)).toBe(tile)
    })

    it("finds nothing off the globe or out of the canvas", () => {
        const camera = cameraOn(positionOf(1), 1)
        expect(picker.pick(camera, 2, 2)).toBeUndefined()
        expect(picker.pick(camera, -1, MIDDLE)).toBeUndefined()
        expect(picker.pick(camera, MIDDLE, canvas.height)).toBeUndefined()
    })

    it("finds nothing over the sea", () => {
        const pacific = new THREE.Vector3(-Math.cos(-0.1) * Math.cos(-2.6), Math.sin(-0.1), Math.cos(-0.1) * Math.sin(-2.6))
        expect(picker.pick(cameraOn(pacific, 20), MIDDLE, MIDDLE)).toBeUndefined()
    })

    it("finds the near side's tile, never the one behind the earth", () => {
        const tile = 4_321
        const camera = cameraOn(positionOf(tile).negate(), 1)
        expect(picker.pick(camera, MIDDLE, MIDDLE)).not.toBe(tile)
    })
})
