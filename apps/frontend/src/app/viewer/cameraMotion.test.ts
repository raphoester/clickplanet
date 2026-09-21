// @vitest-environment jsdom
import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {OrbitControls} from "three/examples/jsm/controls/OrbitControls.js"

/**
 * The render loop draws a frame only when something moved, and it asks
 * OrbitControls whether the camera did — see `drawsFrame` and `startAnimation`
 * in globe.ts. That question has an answer the loop cannot get from its own
 * `update()` alone, and this pins which one, because getting it wrong freezes
 * the globe for the whole of a zoom rather than failing anything.
 */
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
    // OrbitControls applies the wheel inside its own handler: `_handleMouseWheel`
    // calls `update()` there, and that call moves the camera and clears the
    // pending scale. Nothing is left over for the next frame.
    it("has already moved the camera by the time the wheel event returns", () => {
        const {camera, before} = wheeled()

        expect(camera.zoom).toBeGreaterThan(before)
    })

    // Which is why the loop's own `update()` answers "nothing moved" on the very
    // frame the zoom happened. Taking that as the whole truth is what left a
    // zoomed-in globe drawing no frames at all while it was being zoomed.
    it("leaves the next update with nothing to report", () => {
        const {controls} = wheeled()

        expect(controls.update()).toBe(false)
    })

    // `change` is dispatched by whichever `update()` actually moved the camera,
    // whether that is the loop's or the wheel handler's, so it is the signal
    // that survives. The loop reads it alongside `update()`.
    it("dispatches change, which is what the loop reads instead", () => {
        const {changes} = wheeled()

        expect(changes).toBeGreaterThan(0)
    })
})
