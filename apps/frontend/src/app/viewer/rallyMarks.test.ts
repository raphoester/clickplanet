// @vitest-environment jsdom
import {describe, expect, it} from "vitest"
import * as THREE from "three"
import {clockStepOf, clothStyleOf, createRallyMarks, peekOf, RallyCallers} from "./rallyMarks.ts"
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

// A square canvas: at zoom 0.5 the globe is a disc of 200 pixels round its centre.
const VIEW = {width: 800, height: 800}

const FACING = {x: 0, y: 0, z: 1}
const ANTIPODE = {x: 0, y: 0, z: -1}

const rally = (fields: Partial<Rally> = {}): Rally => ({
    id: "r1", accountId: "a1", countryId: "fr", centre: FACING, radius: 0.07, endsAt: 1_000, ...fields,
})

const rules = () => ({rallyCost: 0.5, rallyLastsMs: 1_000})

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
        const marks = createRallyMarks(container, known, rules)

        marks.show([rally()], "fr")
        marks.update(0, cameraOn(facing, 1), VIEW)

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
        const marks = createRallyMarks(container, callers(), rules)

        marks.show([rally({id: "r2", accountId: "a2", countryId: "de"})], "fr")
        marks.update(0, cameraOn(facing, 1), VIEW)

        const pin = container.querySelector<HTMLElement>(".rally-pin")!
        expect(pin.hidden).toBe(false)
        expect(pin.classList.contains("rally-pin--own")).toBe(false)
        expect(container.querySelector<HTMLElement>(".rally-pin-name")!.hidden).toBe(true)
    })

    it("hide a flag behind the globe, and every mark at its rally's end", () => {
        const container = document.createElement("div")
        const marks = createRallyMarks(container, callers(), rules)
        marks.show([rally(), rally({id: "r2", centre: ANTIPODE})], "fr")

        expect(marks.update(0, cameraOn(facing, 1), VIEW)).toBe(true)
        expect(marks.update(1, cameraOn(facing, 1), VIEW)).toBe(false)
        const [front, behind] = container.querySelectorAll<HTMLElement>(".rally-pin")
        expect(front.hidden).toBe(false)
        expect(behind.hidden).toBe(true)

        expect(marks.update(1_000, cameraOn(facing, 1), VIEW)).toBe(true)
        expect(front.hidden).toBe(true)
        expect(marks.update(1_001, cameraOn(facing, 1), VIEW)).toBe(false)
    })

    it("show a flag planted just behind the globe's edge as it rises past it, with no arrow", () => {
        const container = document.createElement("div")
        const marks = createRallyMarks(container, callers(), rules)
        marks.show([rally({centre: {x: 0, y: Math.cos(0.05), z: -Math.sin(0.05)}})], "fr")

        marks.update(0, cameraOn(facing, 0.5), VIEW)

        expect(container.querySelector<HTMLElement>(".rally-pin")!.hidden).toBe(false)
        expect(container.querySelector<HTMLElement>(".bonus-pointer--rally")!.hidden).toBe(true)
    })

    it("point at the newest ring of the page's flag from the globe's edge once its flag is all behind", () => {
        const container = document.createElement("div")
        const marks = createRallyMarks(container, callers(), rules)
        marks.show([rally({centre: {x: 0.6, y: 0, z: -0.8}})], "fr")

        marks.update(0, cameraOn(facing, 0.5), VIEW)

        expect(container.querySelector<HTMLElement>(".rally-pin")!.hidden).toBe(true)
        const pointer = container.querySelector<HTMLElement>(".bonus-pointer--rally")!
        expect(pointer.hidden).toBe(false)
        expect(parseFloat(pointer.style.left)).toBeCloseTo((222 / 400 * 0.5 + 0.5) * 100, 3)
        expect(parseFloat(pointer.style.top)).toBeCloseTo(50, 3)
    })

    it("point at it from the screen's edge when it is off the screen in front", () => {
        const container = document.createElement("div")
        const marks = createRallyMarks(container, callers(), rules)
        marks.show([rally({centre: {x: 0.8, y: 0, z: 0.6}})], "fr")

        marks.update(0, cameraOn(facing, 4), VIEW)

        expect(container.querySelector<HTMLElement>(".rally-pin")!.hidden).toBe(true)
        expect(container.querySelector<HTMLElement>(".bonus-pointer--rally")!.hidden).toBe(false)
    })

    it("point at nothing for another flag's ring", () => {
        const container = document.createElement("div")
        const marks = createRallyMarks(container, callers(), rules)
        marks.show([rally({countryId: "de", centre: ANTIPODE})], "fr")

        marks.update(0, cameraOn(facing, 0.5), VIEW)

        expect(container.querySelector<HTMLElement>(".bonus-pointer--rally")!.hidden).toBe(true)
    })
})

describe("a flag behind the globe", () => {
    const disc = {x: 0, y: 0, radius: 200}

    it("shows what rises past the globe's edge, the disc cut out round its foot", () => {
        const clip = peekOf({x: 0, y: 199 / 400}, disc, VIEW)

        expect(clip).toContain("evenodd")
        expect(clip).toContain("M-200.0 199.0A200.0 200.0 0 1 0 200.0 199.0")
    })

    it("shows nothing once all of it is inside the disc", () => {
        expect(peekOf({x: 0, y: -0.2}, disc, VIEW)).toBeUndefined()
        expect(peekOf({x: 0.3, y: 0}, disc, VIEW)).toBeUndefined()
    })

    it("shows the cloth past the right edge while the pole is still behind", () => {
        expect(peekOf({x: 190 / 400, y: 0}, disc, VIEW)).toBeDefined()
    })
})

describe("the clock", () => {
    it("runs down in steps from whole to nothing", () => {
        expect(clockStepOf(1_000, 1_000)).toBe(360)
        expect(clockStepOf(500, 1_000)).toBe(180)
        expect(clockStepOf(1, 1_000)).toBe(1)
        expect(clockStepOf(0, 1_000)).toBe(0)
        expect(clockStepOf(-5, 1_000)).toBe(0)
        expect(clockStepOf(2_000, 1_000)).toBe(360)
    })
})

describe("the cloth", () => {
    it("covers the cloth with the flag's sprite, cut evenly", () => {
        const style = clothStyleOf({x: 100, y: 50, width: 60, height: 30})

        expect(style.backgroundPosition).toBe("-97.00px -45.00px")
    })
})
