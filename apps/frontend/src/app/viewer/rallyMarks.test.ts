// @vitest-environment jsdom
import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {clothStyleOf, CLOTH, createRallyMarks, RallyCallers} from "./rallyMarks.ts"
import {Rally} from "../../domain/rallies.ts"

function cameraOn(direction: THREE.Vector3, zoom: number) {
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0.1, 10)
    camera.position.copy(direction).setLength(3)
    camera.lookAt(0, 0, 0)
    camera.zoom = zoom
    camera.updateProjectionMatrix()
    camera.updateMatrixWorld()
    return camera
}

// Tile 1 faces the camera; tile 2 is behind the globe.
const positions = Float32Array.from([0, 0, 1, 0, 0, -1])

const rally = (fields: Partial<Rally> = {}): Rally => ({
    id: "r1", accountId: "a1", countryId: "fr", tile: 1, radius: 0.07, endsAt: 1_000, ...fields,
})

const callers = (): RallyCallers & {asked: string[]} => {
    const asked: string[] = []
    return {
        asked,
        ask: (accountId) => asked.push(accountId),
        nameplateOf: (accountId) => accountId === "a1" ? {name: "le professeur", style: {"--author-hue": 200}} : undefined,
    }
}

describe("the rally marks", () => {
    const facing = new THREE.Vector3(0, 0, 1)

    it("plant the rally's flag at its centre, with its caller's name in its color", () => {
        const container = document.createElement("div")
        const known = callers()
        const marks = createRallyMarks(container, known)

        marks.show([rally()], "fr", positions)
        marks.update(0, cameraOn(facing, 1))

        const pin = container.querySelector<HTMLElement>(".rally-pin")!
        expect(pin.hidden).toBe(false)
        expect(pin.classList.contains("rally-pin--own")).toBe(true)
        expect(pin.style.left).toBe("50%")
        expect(pin.style.top).toBe("50%")
        expect(container.querySelector<HTMLElement>(".rally-pin-cloth")!.style.backgroundImage).toContain("url(")
        const name = container.querySelector<HTMLElement>(".rally-pin-name")!
        expect(name.textContent).toBe("le professeur")
        expect(name.style.getPropertyValue("--author-hue")).toBe("200")
        expect(known.asked).toEqual(["a1"])
        marks.dispose()
        expect(container.querySelector(".rally-pin")).toBeNull()
    })

    it("mark another flag's rally too, its name waiting for the nameplates", () => {
        const container = document.createElement("div")
        const marks = createRallyMarks(container, callers())

        marks.show([rally({id: "r2", accountId: "a2", countryId: "de"})], "fr", positions)
        marks.update(0, cameraOn(facing, 1))

        const pin = container.querySelector<HTMLElement>(".rally-pin")!
        expect(pin.hidden).toBe(false)
        expect(pin.classList.contains("rally-pin--own")).toBe(false)
        expect(container.querySelector<HTMLElement>(".rally-pin-name")!.hidden).toBe(true)
    })

    it("hide a flag behind the globe, and every mark at its rally's end", () => {
        const container = document.createElement("div")
        const marks = createRallyMarks(container, callers())
        marks.show([rally(), rally({id: "r2", tile: 2})], "fr", positions)

        expect(marks.update(0, cameraOn(facing, 1))).toBe(false)
        const [front, behind] = container.querySelectorAll<HTMLElement>(".rally-pin")
        expect(front.hidden).toBe(false)
        expect(behind.hidden).toBe(true)

        expect(marks.update(1_000, cameraOn(facing, 1))).toBe(true)
        expect(front.hidden).toBe(true)
        expect(marks.update(1_001, cameraOn(facing, 1))).toBe(false)
    })

    it("point at the newest ring of the page's flag behind the globe", () => {
        const container = document.createElement("div")
        const marks = createRallyMarks(container, callers())
        marks.show([rally({tile: 2})], "fr", positions)

        marks.update(0, cameraOn(facing, 1))

        expect(container.querySelector<HTMLElement>(".bonus-pointer--rally")!.hidden).toBe(false)
    })
})

describe("the cloth", () => {
    it("covers the cloth with the flag's sprite, cut evenly", () => {
        const style = clothStyleOf({x: 100, y: 50, width: 60, height: 30})
        const scale = Math.max(CLOTH.width / 60, CLOTH.height / 30)

        expect(style.backgroundPosition).toBe(
            `${(-100 * scale - (60 * scale - CLOTH.width) / 2).toFixed(2)}px ${(-50 * scale - (30 * scale - CLOTH.height) / 2).toFixed(2)}px`)
    })
})
