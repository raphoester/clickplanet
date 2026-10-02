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

    /**
     * Runs `steps` frames and answers how far round the globe the camera came,
     * in turns. Counted step by step, so more than one turn still reads as more
     * than one turn.
     */
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

/**
 * The spin is paced by `IDLE_FRAME_MS`, and how far it moves between two drawn
 * frames is what decides whether it reads as motion or as a slideshow. That
 * distance is this, and OrbitControls will measure it in two different units
 * depending on what `update` is handed.
 */
describe("the idle spin's speed", () => {
    // Given the time a frame took, the turn is on the clock: `autoRotateSpeed`
    // is turns per minute, so 2 is a turn in thirty seconds whatever is drawing
    // it. This is the unit `SPIN_TURNS_PER_MINUTE` is written in.
    it("is turns per minute, when update is given the time a frame took", () => {
        const {turned} = orbiting()

        // Fifteen seconds' worth, at 60 frames a second.
        expect(turned(900, 1 / 60)).toBeCloseTo(0.5, 2)
    })

    // The same fifteen seconds on a 120Hz display is twice as many frames, and
    // carries the camera exactly as far.
    it("does not depend on how many frames the display offers", () => {
        const {turned} = orbiting()

        expect(turned(1800, 1 / 120)).toBeCloseTo(0.5, 2)
    })

    // Given nothing, OrbitControls falls back to a fixed angle per call, which
    // assumes every display runs at 60. On a 120Hz one the globe went round
    // twice as fast — and so went twice as far between the frames that were
    // drawn, which is what the pacing is trying to keep small.
    it("runs at the display's rate instead when update is given nothing", () => {
        const {turned} = orbiting()

        expect(turned(1800)).toBeCloseTo(1, 2)
    })
})
