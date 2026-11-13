// @vitest-environment jsdom
import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {createTapPointer} from "./tapPointer.ts"

function cameraOn(direction: THREE.Vector3, zoom: number) {
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0.1, 10)
    camera.position.copy(direction).setLength(3)
    camera.lookAt(0, 0, 0)
    camera.zoom = zoom
    camera.updateProjectionMatrix()
    camera.updateMatrixWorld()
    return camera
}

describe("the tap pointer", () => {
    const facing = new THREE.Vector3(0, 0, 1)

    it("stands on the tile's place on the screen", () => {
        const container = document.createElement("div")
        const pointer = createTapPointer(container)
        const root = container.querySelector<HTMLElement>(".tap-pointer")!

        pointer.update(facing.clone(), cameraOn(facing, 1))

        expect(root.hidden).toBe(false)
        expect(root.style.left).toBe("50%")
        expect(root.style.top).toBe("50%")
        expect(root.textContent).toBe("")

        pointer.dispose()
        expect(container.querySelector(".tap-pointer")).toBeNull()
    })

    it("hides while the tile is behind the globe, off the screen, or none", () => {
        const container = document.createElement("div")
        const pointer = createTapPointer(container)
        const root = container.querySelector<HTMLElement>(".tap-pointer")!
        const aside = new THREE.Vector3(Math.sin(0.3), 0, Math.cos(0.3))

        pointer.update(facing.clone().negate(), cameraOn(facing, 1))
        expect(root.hidden).toBe(true)

        pointer.update(aside, cameraOn(facing, 10))
        expect(root.hidden).toBe(true)

        pointer.update(aside, cameraOn(facing, 1))
        expect(root.hidden).toBe(false)

        pointer.update(undefined, cameraOn(facing, 1))
        expect(root.hidden).toBe(true)
    })
})
