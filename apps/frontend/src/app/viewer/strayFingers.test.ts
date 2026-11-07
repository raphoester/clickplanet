// @vitest-environment jsdom
import {afterEach, beforeEach, describe, expect, it} from "vitest"
import * as THREE from "three"
import {OrbitControls} from "three/addons/controls/OrbitControls.js"
import {endStrayFingers} from "./strayFingers.ts"

// The pointers down and the capture jsdom lacks, as a browser keeps them: a pointer that ends loses its capture, and
// releasing one that has ended throws.
function pointersOf(canvas: HTMLCanvasElement) {
    const down = new Set<number>()
    const held = new Set<number>()
    canvas.setPointerCapture = (pointerId: number) => {
        held.add(pointerId)
    }
    canvas.hasPointerCapture = (pointerId: number) => held.has(pointerId)
    canvas.releasePointerCapture = (pointerId: number) => {
        if (!down.has(pointerId)) throw new DOMException("No active pointer with the given id", "NotFoundError")
        held.delete(pointerId)
    }
    return {
        press: (pointerId: number) => down.add(pointerId),
        end: (pointerId: number) => {
            down.delete(pointerId)
            held.delete(pointerId)
        },
    }
}

function touchEvent(type: string, target: EventTarget, onScreen: {touches: number, targetTouches: number}) {
    const event = new TouchEvent(type, {bubbles: true})
    Object.defineProperty(event, "touches", {value: {length: onScreen.touches}})
    Object.defineProperty(event, "targetTouches", {value: {length: onScreen.targetTouches}})
    target.dispatchEvent(event)
}

describe("endStrayFingers", () => {
    let canvas: HTMLCanvasElement
    let camera: THREE.OrthographicCamera
    let controls: OrbitControls
    let lifetime: AbortController
    let pointers: ReturnType<typeof pointersOf>
    let errors: unknown[]
    const onError = (event: ErrorEvent) => errors.push(event.error)

    const pointer = (type: string, pointerId: number, x: number, y: number) => {
        if (type === "pointerdown") pointers.press(pointerId)
        canvas.dispatchEvent(new PointerEvent(type, {pointerId, pointerType: "touch", clientX: x, clientY: y, bubbles: true}))
        if (type === "pointerup") pointers.end(pointerId)
    }

    // A finger pressed on the canvas, alone on the screen with whatever iOS took.
    const press = (pointerId: number, x: number, y: number) => {
        pointer("pointerdown", pointerId, x, y)
        touchEvent("touchstart", canvas, {touches: 1, targetTouches: 1})
    }

    const lift = (pointerId: number, x: number, y: number) => {
        pointer("pointerup", pointerId, x, y)
        touchEvent("touchend", canvas, {touches: 0, targetTouches: 0})
    }

    // The swipe back iOS takes from the page: its pointerdown arrives, and then nothing of it ever again.
    const swipeBackLetGo = (pointerId: number) => {
        press(pointerId, 2, 300)
        pointers.end(pointerId)
    }

    beforeEach(() => {
        canvas = document.createElement("canvas")
        Object.defineProperty(canvas, "clientWidth", {value: 400})
        Object.defineProperty(canvas, "clientHeight", {value: 800})
        document.body.append(canvas)
        pointers = pointersOf(canvas)
        errors = []
        window.addEventListener("error", onError)

        camera = new THREE.OrthographicCamera(-0.5, 0.5, 1, -1, 0.01, 100)
        camera.position.set(0, 0, 5)
        controls = new OrbitControls(camera, canvas)
        lifetime = new AbortController()
        endStrayFingers(canvas, lifetime.signal)
    })

    afterEach(() => {
        lifetime.abort()
        controls.dispose()
        canvas.remove()
        window.removeEventListener("error", onError)
    })

    it("turns the globe under one finger after a swipe back iOS let go of, instead of zooming", () => {
        swipeBackLetGo(1)

        press(2, 200, 400)
        pointer("pointermove", 2, 200, 250)

        expect(camera.zoom).toBe(1)
        expect(camera.position.y).toBeLessThan(-1)
        lift(2, 200, 250)
        expect(errors).toEqual([])
    })

    it("leaves a pinch of two fingers that are both down alone", () => {
        press(1, 100, 400)
        pointer("pointerdown", 2, 300, 400)
        touchEvent("touchstart", canvas, {touches: 2, targetTouches: 2})

        pointer("pointermove", 2, 380, 400)

        expect(camera.zoom).toBeGreaterThan(1)
    })

    it("lets go of the stray once no finger is on the screen, without throwing on its capture", () => {
        const ends: string[] = []
        controls.addEventListener("end", () => ends.push("end"))
        swipeBackLetGo(1)

        touchEvent("touchend", document.body, {touches: 0, targetTouches: 0})

        expect(ends).toEqual(["end"])
        expect(errors).toEqual([])
        press(2, 200, 400)
        pointer("pointermove", 2, 200, 250)
        expect(camera.zoom).toBe(1)
    })

    it("lets go of the stray when the page is put away", () => {
        const ends: string[] = []
        controls.addEventListener("end", () => ends.push("end"))
        swipeBackLetGo(1)

        window.dispatchEvent(new Event("pagehide"))

        expect(ends).toEqual(["end"])
        expect(errors).toEqual([])
    })

    it("lets a finger lifted the usual way go once", () => {
        const ends: string[] = []
        controls.addEventListener("end", () => ends.push("end"))

        press(1, 200, 400)
        lift(1, 200, 400)

        expect(ends).toEqual(["end"])
        expect(errors).toEqual([])
    })
})
