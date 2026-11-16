// @vitest-environment jsdom
import {afterEach, describe, expect, it, vi} from "vitest"
import * as THREE from "three"
import {createRallyPops} from "./rallyPops.ts"

function cameraOn(direction: THREE.Vector3, zoom: number) {
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0.1, 10)
    camera.position.copy(direction).setLength(3)
    camera.lookAt(0, 0, 0)
    camera.zoom = zoom
    camera.updateProjectionMatrix()
    camera.updateMatrixWorld()
    return camera
}

describe("the rally pops", () => {
    const facing = new THREE.Vector3(0, 0, 1)

    afterEach(() => {
        vi.useRealTimers()
    })

    it("rise from the tile clicked, and go after their animation", () => {
        vi.useFakeTimers()
        const container = document.createElement("div")
        const pops = createRallyPops(container)

        pops.pop(facing.clone(), cameraOn(facing, 1), "2×")

        const pop = container.querySelector<HTMLElement>(".rally-pop")!
        expect(pop.textContent).toBe("2×")
        expect(pop.style.left).toBe("50%")
        vi.advanceTimersByTime(1_000)
        expect(container.querySelector(".rally-pop")).toBeNull()
    })

    it("make no pile under a fast thumb, and none off the screen", () => {
        const container = document.createElement("div")
        const pops = createRallyPops(container)
        const camera = cameraOn(facing, 1)

        for (let i = 0; i < 20; i++) pops.pop(facing.clone(), camera, "2×")
        pops.pop(new THREE.Vector3(Math.sin(0.3), 0, Math.cos(0.3)), cameraOn(facing, 10), "2×")

        expect(container.querySelectorAll(".rally-pop")).toHaveLength(8)
        pops.dispose()
        expect(container.querySelectorAll(".rally-pop")).toHaveLength(0)
    })
})
