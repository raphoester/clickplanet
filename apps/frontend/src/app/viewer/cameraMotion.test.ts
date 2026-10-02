// @vitest-environment jsdom
import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {OrbitControls} from "three/examples/jsm/controls/OrbitControls.js"

function orbiting() {
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0.1, 100)
    camera.position.set(0, 0, 5)
    camera.updateProjectionMatrix()

    const element = document.createElement("div")
    document.body.appendChild(element)

    const controls = new OrbitControls(camera, element)
    controls.autoRotate = true
    controls.autoRotateSpeed = 2

    const azimuth = () => Math.atan2(camera.position.x, camera.position.z)

    const turned = (steps: number, delta?: number) => {
        let last = azimuth()
        let total = 0
        for (let i = 0; i < steps; i++) {
            controls.update(delta)
            let moved = azimuth() - last
            while (moved < -Math.PI) moved += 2 * Math.PI
            while (moved > Math.PI) moved -= 2 * Math.PI
            total += moved
            last = azimuth()
        }
        return Math.abs(total) / (2 * Math.PI)
    }

    return {camera, controls, turned}
}

function wheeled() {
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0.1, 100)
    camera.position.set(0, 0, 5)
    camera.zoom = 1
    camera.updateProjectionMatrix()

    const element = document.createElement("div")
    document.body.appendChild(element)

    const controls = new OrbitControls(camera, element)
    controls.minZoom = 0.5
    controls.maxZoom = 50
    controls.enableDamping = true

    let changes = 0
    controls.addEventListener("change", () => {
        changes++
    })

    const before = camera.zoom
    element.dispatchEvent(new WheelEvent("wheel", {deltaY: -120, cancelable: true}))

    return {camera, controls, changes, before, element}
}

describe("a wheel zoom, as the render loop sees it", () => {
    it("has already moved the camera by the time the wheel event returns", () => {
        const {camera, before} = wheeled()

        expect(camera.zoom).toBeGreaterThan(before)
    })

    it("leaves the next update with nothing to report", () => {
        const {controls} = wheeled()

        expect(controls.update()).toBe(false)
    })

    it("dispatches change, which is what the loop reads instead", () => {
        const {changes} = wheeled()

        expect(changes).toBeGreaterThan(0)
    })
})

describe("the idle spin's speed", () => {
    it("is turns per minute, when update is given the time a frame took", () => {
        const {turned} = orbiting()

        expect(turned(900, 1 / 60)).toBeCloseTo(0.5, 2)
    })

    it("does not depend on how many frames the display offers", () => {
        const {turned} = orbiting()

        expect(turned(1800, 1 / 120)).toBeCloseTo(0.5, 2)
    })

    it("runs at the display's rate instead when update is given nothing", () => {
        const {turned} = orbiting()

        expect(turned(1800)).toBeCloseTo(1, 2)
    })
})
